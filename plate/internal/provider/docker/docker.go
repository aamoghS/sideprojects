package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"plate/internal/plans"
	"plate/internal/vm"
)

type Provider struct {
	Image string
}

func New(image string) *Provider {
	if image == "" {
		image = "ubuntu:22.04"
	}
	return &Provider{Image: image}
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
	_, err := p.run(ctx, "rm", "-f", containerName(inst.ID))
	return err
}

func (p *Provider) Sync(ctx context.Context, inst vm.Instance) (vm.Instance, error) {
	name := containerName(inst.ID)
	out, err := p.run(ctx, "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		inst.Status = vm.StatusError
		inst.Error = "missing in docker"
		return inst, nil
	}
	if strings.TrimSpace(out) == "true" {
		inst.Status = vm.StatusRunning
	} else {
		inst.Status = vm.StatusStopped
	}

	ipOut, err := p.run(ctx, "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", name)
	if err == nil {
		inst.IPv4 = strings.TrimSpace(ipOut)
	}
	return inst, nil
}

func containerName(id string) string {
	return "plate-" + id
}

func (p *Provider) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
