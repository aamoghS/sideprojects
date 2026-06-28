package control

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/aamoghS/sideprojects/plate/internal/plans"
	"github.com/aamoghS/sideprojects/plate/internal/provider"
	"github.com/aamoghS/sideprojects/plate/internal/store"
	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

type Plane struct {
	Store    *store.Store
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

	now := time.Now().UTC()
	inst := vm.Instance{
		ID:        newID(),
		Name:      name,
		Plan:      plan.ID,
		Image:     strings.TrimSpace(req.Image),
		Status:    vm.StatusCreating,
		Provider:  p.Provider,
		CreatedAt: now,
		UpdatedAt: now,
	}

	backendID, err := p.Backend.Create(ctx, inst, plan)
	if err != nil {
		inst.Status = vm.StatusError
		inst.Error = err.Error()
		_ = p.Store.Put(inst)
		return inst, err
	}

	inst.BackendID = backendID
	inst.Status = vm.StatusRunning
	inst.UpdatedAt = time.Now().UTC()

	synced, _ := p.Backend.Sync(ctx, inst)
	if synced.Status != "" {
		inst = synced
	}

	if err := p.Store.Put(inst); err != nil {
		return inst, err
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
	inst.UpdatedAt = time.Now().UTC()
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
	inst.UpdatedAt = time.Now().UTC()
	_ = p.Store.Put(inst)
	return inst, nil
}

func (p *Plane) Delete(ctx context.Context, id string) error {
	inst, err := p.Store.Get(id)
	if err != nil {
		return err
	}
	if err := p.Backend.Delete(ctx, inst); err != nil {
		return err
	}
	return p.Store.Delete(id)
}

func newID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
