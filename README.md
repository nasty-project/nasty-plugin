# kubectl-nasty

A `kubectl` plugin for managing [NASty CSI](https://github.com/nasty-project/nasty-csi) volumes. Provides visibility into volumes, snapshots, clones, and health — directly from the command line.

## Screenshot

<img width="1716" height="938" alt="image" src="https://github.com/user-attachments/assets/2076333e-5ea0-4733-9e28-7512464d56f6" />

## Examples

```
❯ kubectl nasty connectivity
Testing NASty connectivity...

... Checking configuration...
✓ Configuration: OK
  URL: wss://10.10.20.100/ws
  API Key: [configured, 43 chars]

... Connecting to NASty...
✓ Connection: OK (0.02s)

... Verifying API access...
  (No default filesystem, checking access...)
✓ API access: OK (0.09s)

... Counting managed volumes...
  Managed volumes: 12
    NFS: 1
    NVMe-oF: 4
    iSCSI: 7
✓ Volume count: OK

All checks passed!

❯ kubectl nasty list
 DATASET                                        │ VOLUME_ID                                │ PROTOCOL │ CAPACITY │ PVC                        │ NAMESPACE  │ TYPE       │ CLONE_SOURCE │ ADOPTABLE
────────────────────────────────────────────────┼──────────────────────────────────────────┼──────────┼──────────┼────────────────────────────┼────────────┼────────────┼──────────────┼───────────
 first/pvc-abfea051-e0dc-4e79-b197-d23c14db1742 │ pvc-abfea051-e0dc-4e79-b197-d23c14db1742 │ iSCSI    │ 50.0Gi   │ ollama                     │ selfhosted │ block      │              │
 first/pvc-a239f9ad-d435-4c0b-96b5-cd81c86205ec │ pvc-a239f9ad-d435-4c0b-96b5-cd81c86205ec │ NVMe-oF  │ 1.0Gi    │ netbox-media               │ net        │ block      │              │
 first/pvc-a6f7e628-4e1d-4e7f-82f1-4daa5b556b78 │ pvc-a6f7e628-4e1d-4e7f-82f1-4daa5b556b78 │ iSCSI    │ 50.0Gi   │ postgres-2                 │ db         │ block      │              │
 first/pvc-633101d2-6745-4bdb-9200-80bcb058942c │ pvc-633101d2-6745-4bdb-9200-80bcb058942c │ iSCSI    │ 5.0Gi    │ ollama-ui                  │ selfhosted │ block      │              │
 first/pvc-d505695a-cd12-4f3c-b47f-d694421fb545 │ pvc-d505695a-cd12-4f3c-b47f-d694421fb545 │ NFS      │ 1000.0Gi │ media                      │ media      │ filesystem │              │ true
 first/pvc-d691fdf5-67e5-49b9-a486-099572d11ae3 │ pvc-d691fdf5-67e5-49b9-a486-099572d11ae3 │ iSCSI    │ 50.0Gi   │ postgres-3                 │ db         │ block      │              │
 first/pvc-350497d7-37bd-4fe1-8325-5e9b8581c7bd │ pvc-350497d7-37bd-4fe1-8325-5e9b8581c7bd │ NVMe-oF  │ 20.0Gi   │ vmsingle-vm                │ o11y       │ block      │              │
 first/pvc-0dac8efa-1397-44a3-9e7e-61f06c594fc4 │ pvc-0dac8efa-1397-44a3-9e7e-61f06c594fc4 │ NVMe-oF  │ 100.0Gi  │ server-volume-vl-server-0  │ o11y       │ block      │              │
 first/pvc-3df37361-bdf9-420f-b8cb-4328e4151db7 │ pvc-3df37361-bdf9-420f-b8cb-4328e4151db7 │ iSCSI    │ 5.0Gi    │ config-qbittorrent-0       │ media      │ block      │              │
 first/pvc-dfcc1361-6ba4-42c0-91f4-64d459d97409 │ pvc-dfcc1361-6ba4-42c0-91f4-64d459d97409 │ iSCSI    │ 50.0Gi   │ postgres-1                 │ db         │ block      │              │
 first/pvc-6af6131d-1145-4c87-b83e-e720c0702450 │ pvc-6af6131d-1145-4c87-b83e-e720c0702450 │ iSCSI    │ 50.0Gi   │ vllm                       │ selfhosted │ block      │              │
 first/pvc-57715c1b-ad46-4ded-81f5-d682388d9ddc │ pvc-57715c1b-ad46-4ded-81f5-d682388d9ddc │ NVMe-oF  │ 2.0Gi    │ dragonfly-data-dragonfly-0 │ db         │ block      │              │

❯ kubectl nasty summary
=== NASty CSI Summary ===

Volumes
  Total: 12    NFS: 1     NVMe-oF: 4     iSCSI: 7

Snapshots
  Total: 0      Attached: 0      Detached: 0

Capacity
  Provisioned: 1.4Ti      Used: 466.4Gi    (33.7%)

Health
  ✓ Healthy: 12
 󱃾 o-homelab/db  ~ ❯
```

## Installation

```bash
kubectl krew install nasty
```

Then use it as:

```bash
kubectl nasty list
```

## Commands

| Command | Description |
|---------|-------------|
| `list` | List all nasty-csi managed volumes |
| `describe <pvc>` | Show detailed information about a volume |
| `status <pvc>` | Show volume status from NASty |
| `summary` | Summary of all nasty-csi managed resources |
| `health` | Check health of all managed volumes |
| `list-snapshots` | List all managed snapshots |
| `list-clones` | List cloned volumes with dependency info |
| `list-orphaned` | Find volumes on NASty without matching PVCs |
| `list-unmanaged` | List volumes not managed by nasty-csi |
| `adopt <path>` | Generate static PV/PVC manifests to adopt an orphaned volume |
| `import <path>` | Import an existing subvolume into nasty-csi management |
| `mark-adoptable` | Mark volumes as adoptable for cross-cluster migration |
| `cleanup` | Delete orphaned volumes from NASty |
| `connectivity` | Test connectivity to NASty |
| `troubleshoot <pvc>` | Diagnose issues with a PVC |
| `dashboard` | Start a local web dashboard for volume overview |

## Examples

```bash
# List all volumes with protocol, size, and PVC binding
kubectl nasty list

# Check if NASty is reachable from the cluster
kubectl nasty connectivity

# Find volumes left behind after PVC deletion
kubectl nasty list-orphaned

# Diagnose why a PVC isn't binding
kubectl nasty troubleshoot my-stuck-pvc

# Adopt an orphaned volume into a new cluster
kubectl nasty adopt storage/pvc-12345 --protocol nfs --server 10.0.0.1
```

## Related

- [NASty](https://github.com/nasty-project/nasty) — bcachefs-based NAS appliance
- [nasty-csi](https://github.com/nasty-project/nasty-csi) — CSI driver
- [nasty-go](https://github.com/nasty-project/nasty-go) — Go client library
- [nasty-chart](https://github.com/nasty-project/nasty-chart) — Helm chart

## License

GPL-3.0
