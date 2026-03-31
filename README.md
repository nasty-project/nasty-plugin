# kubectl-nasty

A `kubectl` plugin for managing [NASty CSI](https://github.com/nasty-project/nasty-csi) volumes. Provides visibility into volumes, snapshots, clones, and health — directly from the command line.

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
