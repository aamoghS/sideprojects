# plate

Minimal VPS control plane in Go. You run `plate serve` on a host; customers (or you) create VMs through an HTTP API and CLI.

```
  CLI / API  -->  plate control plane  -->  docker (dev) or proxmox (real KVM)
```

## Quick start (Docker provider)

Good for learning the control plane on any machine with Docker installed.

```bash
cd plate
chmod +x run.sh          # once
./run.sh                 # build + serve on :8080
./run.sh create --name crawl-1 --plan medium
./run.sh list
```

Optional: `cp plate.env.example plate.env` to set provider, listen addr, proxmox creds.

Manual build:

```bash
go build -o plate ./cmd/plate
./plate serve --provider docker --listen :8080
```

Plans: `tiny`, `small`, `medium`, `large` (CPU/RAM/disk).

State lives in `.plate/vms.json`.

## Real VPS path (Proxmox + KVM)

Run the control plane on a Linux box with [Proxmox VE](https://www.proxmox.com/) installed.

1. Create a Ubuntu cloud-init **template VM** (e.g. VMID `9000`).
2. Set env vars:

```bash
export PLATE_PROXMOX_URL=https://your-host:8006
export PLATE_PROXMOX_USER=root@pam
export PLATE_PROXMOX_PASSWORD=secret
export PLATE_PROXMOX_NODE=pve
export PLATE_PROXMOX_TEMPLATE=9000
export PLATE_PROXMOX_STORAGE=local-lvm
export PLATE_PROXMOX_BRIDGE=vmbr0
export PLATE_PROXMOX_INSECURE=true   # only if using self-signed TLS
```

3. Start the server:

```bash
./plate serve --provider proxmox --listen :8080
./plate create --name customer-1 --plan small
```

Plate clones your template, resizes CPU/RAM/disk, and starts the VM.

## HTTP API

| Method | Path | Action |
|--------|------|--------|
| GET | `/v1/plans` | List plans |
| GET | `/v1/vms` | List VMs |
| POST | `/v1/vms` | Create `{"name":"web-1","plan":"small"}` |
| GET | `/v1/vms/{id}` | Get VM |
| POST | `/v1/vms/{id}/start` | Start |
| POST | `/v1/vms/{id}/stop` | Stop |
| DELETE | `/v1/vms/{id}` | Delete |

## What you still add for a real provider

Plate is the **control plane core**. A public VPS product also needs:

- Public IP allocation (BGP or provider IP pool)
- Reverse DNS, firewall rules per customer
- SSH key injection (cloud-init user-data)
- Billing, accounts, API tokens
- Monitoring, backups, snapshots
- Panel UI

Those layer on top of this API without changing the provider interface.

## Tests

```bash
go test ./...
```
