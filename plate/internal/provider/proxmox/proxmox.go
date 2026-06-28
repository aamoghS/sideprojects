package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aamoghS/sideprojects/plate/internal/plans"
	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

type Config struct {
	URL      string
	User     string
	Password string
	Node     string
	Storage  string
	Bridge   string
	Template int
	Insecure bool
}

func ConfigFromEnv() Config {
	tmpl, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("PLATE_PROXMOX_TEMPLATE")))
	return Config{
		URL:      strings.TrimRight(os.Getenv("PLATE_PROXMOX_URL"), "/"),
		User:     os.Getenv("PLATE_PROXMOX_USER"),
		Password: os.Getenv("PLATE_PROXMOX_PASSWORD"),
		Node:     envOr("PLATE_PROXMOX_NODE", "pve"),
		Storage:  envOr("PLATE_PROXMOX_STORAGE", "local-lvm"),
		Bridge:   envOr("PLATE_PROXMOX_BRIDGE", "vmbr0"),
		Template: tmpl,
		Insecure: os.Getenv("PLATE_PROXMOX_INSECURE") == "true",
	}
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

type Provider struct {
	cfg    Config
	client *http.Client
	ticket string
	csrf   string
}

func New(cfg Config) (*Provider, error) {
	if cfg.URL == "" || cfg.User == "" || cfg.Password == "" {
		return nil, fmt.Errorf("set PLATE_PROXMOX_URL, PLATE_PROXMOX_USER, PLATE_PROXMOX_PASSWORD")
	}
	if cfg.Template <= 0 {
		return nil, fmt.Errorf("set PLATE_PROXMOX_TEMPLATE to a QEMU template VMID (e.g. 9000)")
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.Insecure}
	p := &Provider{
		cfg: cfg,
		client: &http.Client{
			Timeout:   60 * time.Second,
			Transport: tr,
		},
	}
	if err := p.login(context.Background()); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) Name() string { return "proxmox" }

func (p *Provider) Create(ctx context.Context, inst vm.Instance, plan plans.Plan) (string, error) {
	vmid, err := p.nextVMID(ctx)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("newid", strconv.Itoa(vmid))
	form.Set("name", inst.Name)
	form.Set("full", "1")
	form.Set("target", p.cfg.Node)

	path := fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/clone", p.cfg.Node, p.cfg.Template)
	if err := p.postForm(ctx, path, form); err != nil {
		return "", fmt.Errorf("clone template: %w", err)
	}

	if err := p.waitTask(ctx, path); err != nil {
		return "", err
	}

	resize := url.Values{}
	resize.Set("cores", strconv.Itoa(plan.CPU))
	resize.Set("memory", strconv.Itoa(plan.Memory))
	resizePath := fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/config", p.cfg.Node, vmid)
	if err := p.putForm(ctx, resizePath, resize); err != nil {
		return "", fmt.Errorf("resize vm: %w", err)
	}

	if plan.Disk > 0 {
		disk := url.Values{}
		disk.Set("scsi0", fmt.Sprintf("%s:%d", p.cfg.Storage, plan.Disk))
		_ = p.putForm(ctx, resizePath, disk)
	}

	startPath := fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/status/start", p.cfg.Node, vmid)
	if err := p.postForm(ctx, startPath, url.Values{}); err != nil {
		return "", fmt.Errorf("start vm: %w", err)
	}

	return strconv.Itoa(vmid), nil
}

func (p *Provider) Start(ctx context.Context, inst vm.Instance) error {
	path := fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/status/start", p.cfg.Node, inst.BackendID)
	return p.postForm(ctx, path, url.Values{})
}

func (p *Provider) Stop(ctx context.Context, inst vm.Instance) error {
	path := fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/status/stop", p.cfg.Node, inst.BackendID)
	return p.postForm(ctx, path, url.Values{})
}

func (p *Provider) Delete(ctx context.Context, inst vm.Instance) error {
	path := fmt.Sprintf("/api2/json/nodes/%s/qemu/%s", p.cfg.Node, inst.BackendID)
	return p.delete(ctx, path)
}

func (p *Provider) Sync(ctx context.Context, inst vm.Instance) (vm.Instance, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/status/current", p.cfg.Node, inst.BackendID)
	body, err := p.get(ctx, path)
	if err != nil {
		inst.Status = vm.StatusError
		inst.Error = err.Error()
		return inst, nil
	}

	var resp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return inst, err
	}
	switch resp.Data.Status {
	case "running":
		inst.Status = vm.StatusRunning
	default:
		inst.Status = vm.StatusStopped
	}
	return inst, nil
}

func (p *Provider) login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", p.cfg.User)
	form.Set("password", p.cfg.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.URL+"/api2/json/access/ticket", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxmox login: %s", strings.TrimSpace(string(body)))
	}

	var out struct {
		Data struct {
			Ticket              string `json:"ticket"`
			CSRFPreventionToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return err
	}
	p.ticket = out.Data.Ticket
	p.csrf = out.Data.CSRFPreventionToken
	return nil
}

func (p *Provider) nextVMID(ctx context.Context) (int, error) {
	body, err := p.get(ctx, "/api2/json/cluster/nextid")
	if err != nil {
		return 0, err
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, err
	}
	return strconv.Atoi(out.Data)
}

func (p *Provider) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.URL+path, nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: p.ticket})
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("proxmox GET %s: %s", path, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (p *Provider) postForm(ctx context.Context, path string, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: p.ticket})
	if p.csrf != "" {
		req.Header.Set("CSRFPreventionToken", p.csrf)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("proxmox POST %s: %s", path, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Provider) putForm(ctx context.Context, path string, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, p.cfg.URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: p.ticket})
	if p.csrf != "" {
		req.Header.Set("CSRFPreventionToken", p.csrf)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("proxmox PUT %s: %s", path, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Provider) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, p.cfg.URL+path, nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: p.ticket})
	if p.csrf != "" {
		req.Header.Set("CSRFPreventionToken", p.csrf)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("proxmox DELETE %s: %s", path, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Provider) waitTask(ctx context.Context, clonePath string) error {
	_ = clonePath
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		time.Sleep(2 * time.Second)
		return nil
	}
	return fmt.Errorf("proxmox clone timed out")
}
