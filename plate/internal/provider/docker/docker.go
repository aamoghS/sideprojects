package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aamoghS/sideprojects/plate/internal/plans"
	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

type Provider struct {
	Image  string
	DataDir string
}

func New(image, dataDir string) *Provider {
	if image == "" {
		image = "ubuntu:22.04"
	}
	return &Provider{Image: image, DataDir: dataDir}
}

func (p *Provider) Name() string { return "docker" }

func (p *Provider) Create(ctx context.Context, inst vm.Instance, plan plans.Plan) (string, error) {
	name := containerName(inst.ID)
	image := inst.Image
	if image == "" {
		image = p.Image
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--label", "plate.managed=true",
		"--label", "plate.vm.id=" + inst.ID,
		"--memory", fmt.Sprintf("%dm", plan.Memory),
		"--cpus", strconv.Itoa(plan.CPU),
	}

	if plan.Disk > 0 {
		args = append(args, "--storage-opt", fmt.Sprintf("size=%dG", plan.Disk))
	}

	if len(inst.SSHKeys) > 0 {
		keyPath, err := p.writeSSHKeys(inst.ID, inst.SSHKeys)
		if err != nil {
			return "", err
		}
		args = append(args,
			"--mount", fmt.Sprintf("type=bind,src=%s,dst=/root/.ssh/authorized_keys,readonly", keyPath),
		)
	}

	args = append(args, image, "sleep", "infinity")

	out, err := p.run(ctx, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (p *Provider) Start(ctx context.Context, inst vm.Instance) error {
	_, err := p.run(ctx, "start", containerName(inst.ID))
	return err
}

func (p *Provider) Stop(ctx context.Context, inst vm.Instance) error {
	_, err := p.run(ctx, "stop", containerName(inst.ID))
	return err
}

func (p *Provider) Delete(ctx context.Context, inst vm.Instance) error {
	_ = p.RemoveFirewall(ctx, inst)
	snaps, _ := p.SnapshotList(ctx, inst)
	for _, s := range snaps {
		_, _ = p.run(ctx, "rmi", "-f", snapImage(inst.ID, s.ID))
	}
	_ = os.Remove(p.sshKeyPath(inst.ID))
	_, err := p.run(ctx, "rm", "-f", containerName(inst.ID))
	return err
}

func (p *Provider) Sync(ctx context.Context, inst vm.Instance) (vm.Instance, error) {
	name := containerName(inst.ID)
	out, err := p.run(ctx, "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		inst.Status = vm.StatusError
		inst.Error = "missing in docker"
		inst.Health = &vm.Health{OK: false, Detail: "container missing", CheckedAt: time.Now().UTC()}
		return inst, nil
	}
	running := strings.TrimSpace(out) == "true"
	if running {
		inst.Status = vm.StatusRunning
	} else {
		inst.Status = vm.StatusStopped
	}

	ipOut, err := p.run(ctx, "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", name)
	if err == nil {
		inst.IPv4 = strings.TrimSpace(ipOut)
	}

	inst.Health = &vm.Health{
		OK:        running,
		Detail:    healthDetail(running),
		CheckedAt: time.Now().UTC(),
	}
	return inst, nil
}

func (p *Provider) ApplyFirewall(ctx context.Context, inst vm.Instance) error {
	if len(inst.Firewall) == 0 || inst.PublicIPv4 == "" || inst.IPv4 == "" {
		return nil
	}
	if runtime.GOOS != "linux" {
		return nil
	}
	_ = p.RemoveFirewall(ctx, inst)
	for _, rule := range inst.Firewall {
		if rule.Protocol != "tcp" && rule.Protocol != "udp" {
			continue
		}
		args := []string{
			"-t", "nat", "-A", "PREROUTING",
			"-d", inst.PublicIPv4,
			"-p", rule.Protocol,
			"--dport", strconv.Itoa(rule.Port),
			"-j", "DNAT",
			"--to-destination", fmt.Sprintf("%s:%d", inst.IPv4, rule.Port),
		}
		if _, err := exec.CommandContext(ctx, "iptables", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("iptables DNAT port %d: %w", rule.Port, err)
		}
	}
	return nil
}

func (p *Provider) RemoveFirewall(ctx context.Context, inst vm.Instance) error {
	if runtime.GOOS != "linux" || inst.PublicIPv4 == "" || inst.IPv4 == "" {
		return nil
	}
	for _, rule := range inst.Firewall {
		if rule.Protocol != "tcp" && rule.Protocol != "udp" {
			continue
		}
		args := []string{
			"-t", "nat", "-D", "PREROUTING",
			"-d", inst.PublicIPv4,
			"-p", rule.Protocol,
			"--dport", strconv.Itoa(rule.Port),
			"-j", "DNAT",
			"--to-destination", fmt.Sprintf("%s:%d", inst.IPv4, rule.Port),
		}
		_, _ = exec.CommandContext(ctx, "iptables", args...).CombinedOutput()
	}
	return nil
}

func (p *Provider) SnapshotCreate(ctx context.Context, inst vm.Instance, name string) (vm.Snapshot, error) {
	if name == "" {
		name = "snap-" + time.Now().UTC().Format("20060102-150405")
	}
	id := newSnapID()
	tag := snapImage(inst.ID, id)
	_, err := p.run(ctx, "commit", containerName(inst.ID), tag, "--pause")
	if err != nil {
		return vm.Snapshot{}, err
	}
	return vm.Snapshot{ID: id, Name: name, CreatedAt: time.Now().UTC()}, nil
}

func (p *Provider) SnapshotList(ctx context.Context, inst vm.Instance) ([]vm.Snapshot, error) {
	prefix := snapImage(inst.ID, "")
	out, err := p.run(ctx, "images", "--format", "{{.Repository}}:{{.Tag}}", prefix)
	if err != nil {
		return nil, err
	}
	var snaps []vm.Snapshot
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, prefix) {
			continue
		}
		id := strings.TrimPrefix(line, prefix)
		snaps = append(snaps, vm.Snapshot{ID: id, Name: id, CreatedAt: time.Now().UTC()})
	}
	return snaps, nil
}

func (p *Provider) SnapshotRestore(ctx context.Context, inst vm.Instance, snapID string) error {
	tag := snapImage(inst.ID, snapID)
	name := containerName(inst.ID)
	_, _ = p.run(ctx, "stop", name)
	_, _ = p.run(ctx, "rm", name)
	args := []string{"run", "-d", "--name", name,
		"--label", "plate.managed=true",
		"--label", "plate.vm.id=" + inst.ID,
	}
	if len(inst.SSHKeys) > 0 {
		if keyPath, err := p.writeSSHKeys(inst.ID, inst.SSHKeys); err == nil {
			args = append(args, "--mount", fmt.Sprintf("type=bind,src=%s,dst=/root/.ssh/authorized_keys,readonly", keyPath))
		}
	}
	args = append(args, tag, "sleep", "infinity")
	_, err := p.run(ctx, args...)
	return err
}

func (p *Provider) writeSSHKeys(vmID string, keys []string) (string, error) {
	dir := filepath.Join(p.DataDir, "ssh-keys")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := p.sshKeyPath(vmID)
	content := strings.Join(keys, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (p *Provider) sshKeyPath(vmID string) string {
	return filepath.Join(p.DataDir, "ssh-keys", vmID+".pub")
}

func containerName(id string) string { return "plate-" + id }

func snapImage(vmID, snapID string) string {
	return "plate-snap-" + vmID + ":" + snapID
}

func newSnapID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func healthDetail(running bool) string {
	if running {
		return "container running"
	}
	return "container stopped"
}

func (p *Provider) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
