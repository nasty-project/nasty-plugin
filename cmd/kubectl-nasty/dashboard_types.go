package main

// Type aliases for types from nasty-go/dashboard.
// Keeps command files clean — they reference VolumeInfo etc. without package qualification.

import "github.com/nasty-project/nasty-go/dashboard"

// Subvolume type constants (matches nasty-go SubvolumeType field values).
const subvolumeTypeBlock = "block"

type (
	VolumeInfo             = dashboard.VolumeInfo
	SnapshotInfo           = dashboard.SnapshotInfo
	CloneInfo              = dashboard.CloneInfo
	UnmanagedVolume        = dashboard.UnmanagedVolume
	HealthStatus           = dashboard.HealthStatus
	VolumeHealth           = dashboard.VolumeHealth
	HealthReport           = dashboard.HealthReport
	HealthSummary          = dashboard.HealthSummary
	K8sVolumeBinding       = dashboard.K8sVolumeBinding
	K8sEnrichmentResult    = dashboard.K8sEnrichmentResult
	VolumeDetails          = dashboard.VolumeDetails
	NFSShareDetails        = dashboard.NFSShareDetails
	NVMeOFSubsystemDetails = dashboard.NVMeOFSubsystemDetails
	SMBShareDetails        = dashboard.SMBShareDetails
	ISCSITargetDetails     = dashboard.ISCSITargetDetails
	MetricsSummary         = dashboard.MetricsSummary
	DashboardData          = dashboard.Data
	SummaryData            = dashboard.SummaryData
	PaginationParams       = dashboard.PaginationParams
	PaginatedVolumes       = dashboard.PaginatedVolumes
	PaginatedSnapshots     = dashboard.PaginatedSnapshots
	PaginatedClones        = dashboard.PaginatedClones
	PaginatedUnmanaged     = dashboard.PaginatedUnmanaged
)
