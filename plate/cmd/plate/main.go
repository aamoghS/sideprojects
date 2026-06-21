package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"plate/internal/api"
	"plate/internal/control"
	"plate/internal/plate"
	"plate/internal/store"
	"plate/internal/vm"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		runServe(os.Args[2:])
	case "plans":
		runPlans(os.Args[2:])
	case "list":
		runList(os.Args[2:])
	case "create":
		runCreate(os.Args[2:])
	case "start":
		runAction("start", os.Args[2:])
	case "stop":
		runAction("stop", os.Args[2:])
	case "delete":
		runDelete(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`plate — minimal VPS control plane in Go

Usage:
  plate serve   [--listen :8080] [--provider docker|proxmox] [--data .plate]
  plate plans   [--api http://127.0.0.1:8080]
  plate list    [--api ...]
  plate create  --name web-1 [--plan small] [--api ...]
  plate start   --id <vm-id> [--api ...]
  plate stop    --id <vm-id> [--api ...]
  plate delete  --id <vm-id> [--api ...]

Providers:
  docker    dev/homelab via Docker (default)
  proxmox   real KVM VMs via Proxmox API (set PLATE_PROXMOX_* env vars)

Examples:
  plate serve --provider docker
  plate create --name crawl-1 --plan medium
  plate list
`)
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", ":8080", "HTTP listen address")
	providerName := fs.String("provider", "docker", "backend: docker or proxmox")
	dataDir := fs.String("data", ".plate", "state directory")
	dockerImage := fs.String("docker-image", "ubuntu:22.04", "default image for docker provider")
	_ = fs.Parse(args)

	backend, name, err := plate.OpenBackend(*providerName, *dockerImage)
	if err != nil {
		fatal(err)
	}

	st, err := store.Open(*dataDir)
	if err != nil {
		fatal(err)
	}

	plane := &control.Plane{Store: st, Backend: backend, Provider: name}
	srv := &api.Server{Plane: plane}

	fmt.Printf("plate listening on %s (provider=%s, data=%s)\n", *listen, name, *dataDir)
	if err := http.ListenAndServe(*listen, srv.Handler()); err != nil {
		fatal(err)
	}
}

func runPlans(args []string) {
	apiURL := parseAPI(args)
	var out any
	if err := apiGET(apiURL+"/v1/plans", &out); err != nil {
		fatal(err)
	}
	printJSON(out)
}

func runList(args []string) {
	apiURL := parseAPI(args)
	var out []vm.Instance
	if err := apiGET(apiURL+"/v1/vms", &out); err != nil {
		fatal(err)
	}
	if len(out) == 0 {
		fmt.Println("no vms")
		return
	}
	for _, v := range out {
		fmt.Printf("%s  %-12s  %-8s  plan=%-6s  provider=%s  ip=%s\n", v.ID, v.Name, v.Status, v.Plan, v.Provider, v.IPv4)
	}
}

func runCreate(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	apiURL := fs.String("api", "http://127.0.0.1:8080", "control plane URL")
	name := fs.String("name", "", "vm name")
	plan := fs.String("plan", "small", "plan id")
	image := fs.String("image", "", "optional image override")
	_ = fs.Parse(args)

	if *name == "" {
		fatal(fmt.Errorf("--name is required"))
	}

	req := vm.CreateRequest{Name: *name, Plan: *plan, Image: *image}
	body, _ := json.Marshal(req)
	resp, err := http.Post(*apiURL+"/v1/vms", "application/json", bytes.NewReader(body))
	if err != nil {
		fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fatal(fmt.Errorf("%s", strings.TrimSpace(string(data))))
	}
	printJSON(json.RawMessage(data))
}

func runAction(action string, args []string) {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	apiURL := fs.String("api", "http://127.0.0.1:8080", "control plane URL")
	id := fs.String("id", "", "vm id")
	_ = fs.Parse(args)
	if *id == "" {
		fatal(fmt.Errorf("--id is required"))
	}
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/vms/%s/%s", *apiURL, *id, action), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fatal(fmt.Errorf("%s", strings.TrimSpace(string(data))))
	}
	printJSON(json.RawMessage(data))
}

func runDelete(args []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	apiURL := fs.String("api", "http://127.0.0.1:8080", "control plane URL")
	id := fs.String("id", "", "vm id")
	_ = fs.Parse(args)
	if *id == "" {
		fatal(fmt.Errorf("--id is required"))
	}
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/vms/%s", *apiURL, *id), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		fatal(fmt.Errorf("%s", strings.TrimSpace(string(data))))
	}
	fmt.Println("deleted")
}

func parseAPI(args []string) string {
	fs := flag.NewFlagSet("cmd", flag.ExitOnError)
	apiURL := fs.String("api", "http://127.0.0.1:8080", "control plane URL")
	_ = fs.Parse(args)
	return strings.TrimRight(*apiURL, "/")
}

func apiGET(url string, out any) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s", strings.TrimSpace(string(data)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(data))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
