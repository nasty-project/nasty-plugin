package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	nastyapi "github.com/nasty-project/nasty-go"
	"github.com/nasty-project/nasty-go/dashboard"
	"k8s.io/klog/v2"
)

// rpcTiming records the duration of a single RPC call.
type rpcTiming struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration_ms"`
	Error    string        `json:"error,omitempty"`
}

// fetchTimings records all RPC call durations for a single data fetch cycle.
type fetchTimings struct {
	StartedAt time.Time    `json:"started_at"`
	Total     time.Duration `json:"total_ms"`
	Calls     []rpcTiming  `json:"calls"`
}

// cachedData holds the fetched dashboard data along with timing info.
type cachedData struct {
	volumes   []VolumeInfo
	snapshots []SnapshotInfo
	clones    []CloneInfo
	unmanaged []UnmanagedVolume
	k8sData   *K8sEnrichmentResult
	summary   SummaryData
	timings   *fetchTimings
	fetchedAt time.Time
}

// dashboardCache provides a single shared NASty client and a short-TTL data cache.
type dashboardCache struct {
	cfg       *connectionConfig
	pool      string
	clusterID string
	cacheTTL  time.Duration

	mu     sync.RWMutex
	client *NAStyClient
	data   *cachedData

	// fetchMu prevents multiple concurrent fetches
	fetchMu sync.Mutex
}

func newDashboardCache(cfg *connectionConfig, pool, clusterID string) *dashboardCache {
	return &dashboardCache{
		cfg:       cfg,
		pool:      pool,
		clusterID: clusterID,
		cacheTTL:  5 * time.Second,
	}
}

// getClient returns the shared client, reconnecting if needed.
func (dc *dashboardCache) getClient(ctx context.Context) (nastyapi.ClientInterface, error) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.client != nil {
		return dc.client, nil
	}

	client, err := connectToNASty(ctx, dc.cfg)
	if err != nil {
		return nil, err
	}
	dc.client = client
	return client, nil
}

// resetClient closes the current client and forces reconnection on next use.
func (dc *dashboardCache) resetClient() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.client != nil {
		dc.client.Close()
		dc.client = nil
	}
}

// close shuts down the shared client.
func (dc *dashboardCache) close() {
	dc.resetClient()
}

// getData returns cached data if fresh, otherwise fetches new data.
func (dc *dashboardCache) getData(ctx context.Context) (*cachedData, error) {
	dc.mu.RLock()
	cached := dc.data
	dc.mu.RUnlock()

	if cached != nil && time.Since(cached.fetchedAt) < dc.cacheTTL {
		return cached, nil
	}

	return dc.fetchData(ctx)
}

// fetchData fetches all dashboard data with per-call timing.
func (dc *dashboardCache) fetchData(ctx context.Context) (*cachedData, error) {
	// Prevent concurrent fetches — second caller gets the result from the first
	dc.fetchMu.Lock()
	defer dc.fetchMu.Unlock()

	// Check again after acquiring lock — another goroutine may have just fetched
	dc.mu.RLock()
	cached := dc.data
	dc.mu.RUnlock()
	if cached != nil && time.Since(cached.fetchedAt) < dc.cacheTTL {
		return cached, nil
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	client, err := dc.getClient(fetchCtx)
	if err != nil {
		// Connection failed — reset so next attempt retries
		dc.resetClient()
		return nil, fmt.Errorf("failed to connect to NASty: %w", err)
	}

	timings := &fetchTimings{StartedAt: time.Now()}
	data := &cachedData{}

	// --- Fetch managed subvolumes (shared across volumes, snapshots, clones, health) ---
	var managedSubvols []nastyapi.Subvolume
	managedSubvols, err = timedCall(timings, "FindSubvolumesByProperty", func() ([]nastyapi.Subvolume, error) {
		return client.FindSubvolumesByProperty(fetchCtx, nastyapi.PropertyManagedBy, nastyapi.ManagedByValue, "")
	})
	if err != nil {
		klog.Warningf("Failed to fetch managed subvolumes: %v", err)
		dc.resetClient()
		return nil, fmt.Errorf("failed to fetch managed subvolumes: %w", err)
	}

	// --- Extract volumes ---
	data.volumes = dashboard.FilterByClusterID(dashboard.ExtractVolumes(managedSubvols), dc.clusterID)

	// --- Fetch snapshots (per filesystem) ---
	filesystems := make(map[string]struct{})
	filteredSubvols := managedSubvols
	if dc.clusterID != "" {
		filteredSubvols = filterManagedSubvolsByCluster(managedSubvols, dc.clusterID)
	}
	for _, sv := range filteredSubvols {
		if sv.Properties[nastyapi.PropertyCSIVolumeName] != "" {
			filesystems[sv.Filesystem] = struct{}{}
		}
	}

	var allSnaps []nastyapi.Snapshot
	for fs := range filesystems {
		fsName := fs
		snaps, snapErr := timedCall(timings, "ListSnapshots("+fsName+")", func() ([]nastyapi.Snapshot, error) {
			return client.ListSnapshots(fetchCtx, fsName)
		})
		if snapErr != nil {
			klog.Warningf("Failed to list snapshots in %s: %v", fsName, snapErr)
			continue
		}
		allSnaps = append(allSnaps, snaps...)
	}
	data.snapshots = matchSnapshotsToVolumes(allSnaps, filteredSubvols)

	// --- Clones (empty for bcachefs, no API call needed) ---
	data.clones = []CloneInfo{}

	// --- Fetch share resources in parallel for health checking ---
	var (
		nfsShares      []nastyapi.NFSShare
		nvmeSubsystems []nastyapi.NVMeOFSubsystem
		smbShares      []nastyapi.SMBShare
		iscsiTargets   []nastyapi.ISCSITarget
		wg             sync.WaitGroup
		nfsErr, nvmeErr, smbErr, iscsiErr error
	)

	wg.Add(4)
	go func() {
		defer wg.Done()
		nfsShares, nfsErr = timedCall(timings, "ListNFSShares", func() ([]nastyapi.NFSShare, error) {
			return client.ListNFSShares(fetchCtx)
		})
	}()
	go func() {
		defer wg.Done()
		nvmeSubsystems, nvmeErr = timedCall(timings, "ListNVMeOFSubsystems", func() ([]nastyapi.NVMeOFSubsystem, error) {
			return client.ListNVMeOFSubsystems(fetchCtx)
		})
	}()
	go func() {
		defer wg.Done()
		smbShares, smbErr = timedCall(timings, "ListSMBShares", func() ([]nastyapi.SMBShare, error) {
			return client.ListSMBShares(fetchCtx)
		})
	}()
	go func() {
		defer wg.Done()
		iscsiTargets, iscsiErr = timedCall(timings, "ListISCSITargets", func() ([]nastyapi.ISCSITarget, error) {
			return client.ListISCSITargets(fetchCtx)
		})
	}()
	wg.Wait()

	if nfsErr != nil {
		klog.Warningf("Failed to list NFS shares: %v", nfsErr)
	}
	if nvmeErr != nil {
		klog.Warningf("Failed to list NVMe-oF subsystems: %v", nvmeErr)
	}
	if smbErr != nil {
		klog.Warningf("Failed to list SMB shares: %v", smbErr)
	}
	if iscsiErr != nil {
		klog.Warningf("Failed to list iSCSI targets: %v", iscsiErr)
	}

	// --- Annotate health using pre-fetched data (no extra API calls) ---
	healthMaps := dashboard.BuildHealthMapsFromData(nfsShares, smbShares, nvmeSubsystems, iscsiTargets)
	dashboard.AnnotateHealthFromMaps(data.volumes, managedSubvols, healthMaps)

	// --- Fetch unmanaged volumes if pool configured ---
	if dc.pool != "" {
		allSubvols, listErr := timedCall(timings, "ListAllSubvolumes", func() ([]nastyapi.Subvolume, error) {
			return client.ListAllSubvolumes(fetchCtx, dc.pool)
		})
		if listErr != nil {
			klog.Warningf("Failed to list all subvolumes: %v", listErr)
		} else {
			data.unmanaged = buildUnmanagedList(allSubvols, managedSubvols, nfsShares)
		}
	}

	// --- K8s enrichment (has its own 5s timeout) ---
	k8sStart := time.Now()
	data.k8sData = enrichWithK8sData(fetchCtx, false)
	timings.Calls = append(timings.Calls, rpcTiming{
		Name:     "K8sEnrichment",
		Duration: time.Since(k8sStart),
	})

	if data.k8sData.Available {
		for i := range data.volumes {
			if binding := dashboard.MatchK8sBinding(data.k8sData.Bindings, data.volumes[i].Dataset, data.volumes[i].VolumeID); binding != nil {
				data.volumes[i].K8s = binding
			}
		}
	}

	// --- Calculate summary ---
	data.summary = dashboard.CalculateSummary(data.volumes, data.snapshots, data.clones)

	timings.Total = time.Since(timings.StartedAt)
	data.timings = timings
	data.fetchedAt = time.Now()

	// Store in cache
	dc.mu.Lock()
	dc.data = data
	dc.mu.Unlock()

	klog.V(2).Infof("Dashboard data fetched in %v (%d RPC calls)", timings.Total, len(timings.Calls))

	return data, nil
}

// timedCall wraps an RPC call with timing measurement.
// Thread-safe: appends to timings under a mutex.
func timedCall[T any](timings *fetchTimings, name string, fn func() (T, error)) (T, error) {
	start := time.Now()
	result, err := fn()
	duration := time.Since(start)

	t := rpcTiming{
		Name:     name,
		Duration: duration,
	}
	if err != nil {
		t.Error = err.Error()
	}

	// fetchTimings is only appended to during a single fetchData call which holds fetchMu,
	// but share list calls run concurrently, so we need synchronization.
	// Use a simple approach: the timings slice is append-safe with this pattern since
	// Go slices are not safe for concurrent append. Let's add a mutex.
	timingsMu.Lock()
	timings.Calls = append(timings.Calls, t)
	timingsMu.Unlock()

	return result, err
}

var timingsMu sync.Mutex

// filterManagedSubvolsByCluster filters subvolumes by cluster ID.
func filterManagedSubvolsByCluster(subvols []nastyapi.Subvolume, clusterID string) []nastyapi.Subvolume {
	if clusterID == "" {
		return subvols
	}
	filtered := make([]nastyapi.Subvolume, 0, len(subvols))
	for i := range subvols {
		prop := subvols[i].Properties[nastyapi.PropertyClusterID]
		if prop == "" || prop == clusterID {
			filtered = append(filtered, subvols[i])
		}
	}
	return filtered
}

// matchSnapshotsToVolumes matches snapshots to managed subvolumes.
func matchSnapshotsToVolumes(snaps []nastyapi.Snapshot, subvols []nastyapi.Subvolume) []SnapshotInfo {
	type subvolMeta struct {
		volumeID string
		protocol string
	}
	managedSubvols := make(map[string]subvolMeta)
	for _, sv := range subvols {
		volumeID := sv.Properties[nastyapi.PropertyCSIVolumeName]
		protocol := sv.Properties[nastyapi.PropertyProtocol]
		if volumeID != "" {
			key := sv.Filesystem + "/" + sv.Name
			managedSubvols[key] = subvolMeta{volumeID: volumeID, protocol: protocol}
		}
	}

	var snapshots []SnapshotInfo
	for _, snap := range snaps {
		subvolKey := snap.Filesystem + "/" + snap.Subvolume
		meta, ok := managedSubvols[subvolKey]
		if !ok {
			continue
		}
		snapshots = append(snapshots, SnapshotInfo{
			Name:          snap.Name,
			SourceVolume:  meta.volumeID,
			SourceDataset: subvolKey,
			Protocol:      meta.protocol,
			Type:          "attached",
		})
	}
	return snapshots
}

// buildUnmanagedList builds the unmanaged volume list from pre-fetched data.
func buildUnmanagedList(allSubvols, managedSubvols []nastyapi.Subvolume, nfsShares []nastyapi.NFSShare) []UnmanagedVolume {
	managedIDs := make(map[string]bool, len(managedSubvols))
	for i := range managedSubvols {
		managedIDs[managedSubvols[i].Filesystem+"/"+managedSubvols[i].Name] = true
	}

	nfsShareByPath := make(map[string]*nastyapi.NFSShare, len(nfsShares))
	for i := range nfsShares {
		nfsShareByPath[nfsShares[i].Path] = &nfsShares[i]
	}

	var volumes []UnmanagedVolume
	for i := range allSubvols {
		sv := &allSubvols[i]
		svID := sv.Filesystem + "/" + sv.Name
		if managedIDs[svID] {
			continue
		}

		vol := UnmanagedVolume{
			Dataset: svID,
			Name:    sv.Name,
			Type:    sv.SubvolumeType,
		}
		if sv.UsedBytes != nil {
			vol.SizeBytes = int64(*sv.UsedBytes)
			vol.Size = dashboard.FormatBytes(vol.SizeBytes)
		}
		if share, ok := nfsShareByPath[sv.Path]; ok {
			vol.Protocol = "nfs"
			vol.NFSShareID = share.ID
			vol.NFSSharePath = share.Path
		} else if sv.SubvolumeType == "block" {
			vol.Protocol = "block"
		}
		volumes = append(volumes, vol)
	}
	return volumes
}
