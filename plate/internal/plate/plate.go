package plate

import (
	"fmt"
	"strings"

	"plate/internal/provider"
	"plate/internal/provider/docker"
	"plate/internal/provider/proxmox"
)

func OpenBackend(name, dockerImage string) (provider.Backend, string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "docker":
		return docker.New(dockerImage), "docker", nil
	case "proxmox":
		p, err := proxmox.New(proxmox.ConfigFromEnv())
		if err != nil {
			return nil, "", err
		}
		return p, "proxmox", nil
	default:
		return nil, "", fmt.Errorf("unknown provider %q (docker or proxmox)", name)
	}
}
