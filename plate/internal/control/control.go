package control

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/aamoghS/sideprojects/plate/internal/accounts"
	"github.com/aamoghS/sideprojects/plate/internal/ippool"
	"github.com/aamoghS/sideprojects/plate/internal/plans"
	"github.com/aamoghS/sideprojects/plate/internal/provider"
	"github.com/aamoghS/sideprojects/plate/internal/store"
	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

type Plane struct {
	Store    *store.Store
	Accounts *accounts.Store
	IPPool   *ippool.Pool
	Backend  provider.Backend
	Provider string
}

func (p *Plane) List(ctx context.Context) ([]vm.Instance, error) {
	items, err := p.Store.List()
	if err != nil {
		return nil, err
	}
	for i := range items {
		synced, err := p.Backend.Sync(ctx, items[i])
		if err == nil {
			items[i] = synced
			p.refreshSnapshots(ctx, &items[i])
			_ = p.Store.Put(items[i])
		}
	}
	return items, nil
}

func (p *Plane) Get(ctx context.Context, id string) (vm.Instance, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return inst, err
	}
	synced, syncErr := p.Backend.Sync(ctx, inst)
	if syncErr == nil {
		inst = synced
		p.refreshSnapshots(ctx, &inst)
		_ = p.Store.Put(inst)
	}
	return inst, nil
}

func (p *Plane) Create(ctx context.Context, req vm.CreateRequest) (vm.Instance, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return vm.Instance{}, fmt.Errorf("name is required")
	}
	planID := strings.TrimSpace(req.Plan)
	if planID == "" {
		planID = "small"
	}
	plan, err := plans.Get(planID)
	if err != nil {
		return vm.Instance{}, err
	}

	firewall := req.Firewall
	if len(firewall) == 0 {
		firewall = vm.DefaultFirewall()
	}

	now := time.Now().UTC()
	inst := vm.Instance{
		ID:        newID(),
		Name:      name,
		Plan:      plan.ID,
		Image:     strings.TrimSpace(req.Image),
		Status:    vm.StatusCreating,
		Provider:  p.Provider,
		SSHKeys:   req.SSHKeys,
		Hostname:  strings.TrimSpace(req.Hostname),
		Firewall:  firewall,
		AccountID: strings.TrimSpace(req.AccountID),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if inst.Hostname == "" {
		inst.Hostname = name + ".plate.local"
	}

	if p.IPPool != nil {
		pool, _, _ := p.IPPool.List()
		if len(pool) > 0 {
			ip, err := p.IPPool.Assign(inst.ID)
			if err != nil {
				return vm.Instance{}, err
			}
			inst.PublicIPv4 = ip
		}
	}

	backendID, err := p.Backend.Create(ctx, inst, plan)
	if err != nil {
		if p.IPPool != nil {
			p.IPPool.Release(inst.ID)
		}
		inst.Status = vm.StatusError
		inst.Error = err.Error()
		_ = p.Store.Put(inst)
		return inst, err
	}

	inst.BackendID = backendID
	inst.Status = vm.StatusRunning
	started := time.Now().UTC()
	inst.StartedAt = &started
	inst.UpdatedAt = started

	publicIP := inst.PublicIPv4
	firewallRules := inst.Firewall
	hostname := inst.Hostname
	sshKeys := inst.SSHKeys
	accountID := inst.AccountID
	synced, _ := p.Backend.Sync(ctx, inst)
	if synced.Status != "" {
		inst = synced
		inst.PublicIPv4 = publicIP
		inst.Firewall = firewallRules
		inst.Hostname = hostname
		inst.SSHKeys = sshKeys
		inst.AccountID = accountID
		inst.BackendID = backendID
	}

	_ = p.Backend.ApplyFirewall(ctx, inst)

	if err := p.Store.Put(inst); err != nil {
		return inst, err
	}

	if p.Accounts != nil && inst.AccountID != "" {
		_ = p.Accounts.RecordUsage(accounts.UsageRecord{
			AccountID: inst.AccountID,
			VMID:      inst.ID,
			VMName:    inst.Name,
			Plan:      inst.Plan,
			CreatedAt: inst.CreatedAt,
		})
	}

	return inst, nil
}

func (p *Plane) Start(ctx context.Context, id string) (vm.Instance, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return inst, err
	}
	if err := p.Backend.Start(ctx, inst); err != nil {
		return inst, err
	}
	inst.Status = vm.StatusRunning
	now := time.Now().UTC()
	inst.StartedAt = &now
	inst.UpdatedAt = now
	synced, _ := p.Backend.Sync(ctx, inst)
	if synced.Status != "" {
		inst.Status = synced.Status
		inst.Health = synced.Health
		inst.IPv4 = synced.IPv4
	}
	_ = p.Store.Put(inst)
	return inst, nil
}

func (p *Plane) Stop(ctx context.Context, id string) (vm.Instance, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return inst, err
	}
	if err := p.Backend.Stop(ctx, inst); err != nil {
		return inst, err
	}
	inst.Status = vm.StatusStopped
	now := time.Now().UTC()
	inst.UpdatedAt = now
	if p.Accounts != nil && inst.AccountID != "" && inst.StartedAt != nil {
		hours := now.Sub(*inst.StartedAt).Hours()
		_ = p.Accounts.UpdateUsageHours(inst.ID, hours, now)
	}
	inst.StartedAt = nil
	synced, _ := p.Backend.Sync(ctx, inst)
	if synced.Health != nil {
		inst.Health = synced.Health
	}
	_ = p.Store.Put(inst)
	return inst, nil
}

func (p *Plane) Delete(ctx context.Context, id string) error {
	inst, err := p.Store.Get(id)
	if err != nil {
		return err
	}
	if p.Accounts != nil && inst.AccountID != "" {
		if inst.StartedAt != nil {
			hours := time.Now().UTC().Sub(*inst.StartedAt).Hours()
			_ = p.Accounts.UpdateUsageHours(inst.ID, hours, time.Now().UTC())
		}
	}
	_ = p.Backend.RemoveFirewall(ctx, inst)
	if err := p.Backend.Delete(ctx, inst); err != nil {
		return err
	}
	if p.IPPool != nil {
		p.IPPool.Release(id)
	}
	return p.Store.Delete(id)
}

func (p *Plane) UpdateFirewall(ctx context.Context, id string, rules []vm.FirewallRule) (vm.Instance, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return inst, err
	}
	_ = p.Backend.RemoveFirewall(ctx, inst)
	inst.Firewall = rules
	inst.UpdatedAt = time.Now().UTC()
	if err := p.Backend.ApplyFirewall(ctx, inst); err != nil {
		return inst, err
	}
	if err := p.Store.Put(inst); err != nil {
		return inst, err
	}
	return inst, nil
}

func (p *Plane) UpdateHostname(ctx context.Context, id, hostname string) (vm.Instance, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return inst, err
	}
	inst.Hostname = strings.TrimSpace(hostname)
	inst.UpdatedAt = time.Now().UTC()
	if err := p.Store.Put(inst); err != nil {
		return inst, err
	}
	return inst, nil
}

func (p *Plane) CreateSnapshot(ctx context.Context, id, name string) (vm.Snapshot, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return vm.Snapshot{}, err
	}
	snap, err := p.Backend.SnapshotCreate(ctx, inst, name)
	if err != nil {
		return vm.Snapshot{}, err
	}
	inst.Snapshots = append(inst.Snapshots, snap)
	inst.UpdatedAt = time.Now().UTC()
	_ = p.Store.Put(inst)
	return snap, nil
}

func (p *Plane) ListSnapshots(ctx context.Context, id string) ([]vm.Snapshot, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return nil, err
	}
	snaps, err := p.Backend.SnapshotList(ctx, inst)
	if err != nil {
		return nil, err
	}
	inst.Snapshots = snaps
	_ = p.Store.Put(inst)
	return snaps, nil
}

func (p *Plane) RestoreSnapshot(ctx context.Context, id, snapID string) (vm.Instance, error) {
	inst, err := p.Store.Get(id)
	if err != nil {
		return inst, err
	}
	if err := p.Backend.SnapshotRestore(ctx, inst, snapID); err != nil {
		return inst, err
	}
	synced, _ := p.Backend.Sync(ctx, inst)
	if synced.Status != "" {
		inst = synced
	}
	inst.UpdatedAt = time.Now().UTC()
	_ = p.Store.Put(inst)
	return inst, nil
}

func (p *Plane) IPPoolStatus() (ippool.Status, error) {
	if p.IPPool == nil {
		return ippool.Status{}, nil
	}
	return p.IPPool.Status()
}

func (p *Plane) refreshSnapshots(ctx context.Context, inst *vm.Instance) {
	snaps, err := p.Backend.SnapshotList(ctx, *inst)
	if err == nil {
		inst.Snapshots = snaps
	}
}

func newID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
