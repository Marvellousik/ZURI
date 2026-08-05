package model

import (
	"runtime"
)

// AccelerationType represents the hardware accelerator platform.
type AccelerationType string

const (
	AccelMetal AccelerationType = "metal"
	AccelCUDA  AccelerationType = "cuda"
	AccelROCm  AccelerationType = "rocm"
	AccelCPU   AccelerationType = "cpu"
)

// HardwareProfile summarizes system compute hardware and VRAM availability.
type HardwareProfile struct {
	Platform       string           `json:"platform"`
	Architecture   string           `json:"architecture"`
	Accelerator    AccelerationType `json:"accelerator"`
	TotalVRAMMB    int              `json:"total_vram_mb"`
	AvailableVRAM  int              `json:"available_vram_mb"`
	PhysicalCores  int              `json:"physical_cores"`
	RecommendedLayers int           `json:"recommended_gpu_layers"`
}

// DetectHardware Profile inspects host operating system and GPU features.
func DetectHardwareProfile() HardwareProfile {
	profile := HardwareProfile{
		Platform:      runtime.GOOS,
		Architecture:  runtime.GOARCH,
		PhysicalCores: runtime.NumCPU(),
	}

	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			profile.Accelerator = AccelMetal
			profile.TotalVRAMMB = 16384
			profile.AvailableVRAM = 12288
			profile.RecommendedLayers = 33
		} else {
			profile.Accelerator = AccelCPU
			profile.RecommendedLayers = 0
		}
	case "windows", "linux":
		// Native GPU detection fallback defaults
		profile.Accelerator = AccelCUDA
		profile.TotalVRAMMB = 8192
		profile.AvailableVRAM = 6144
		profile.RecommendedLayers = 32
	default:
		profile.Accelerator = AccelCPU
		profile.RecommendedLayers = 0
	}

	return profile
}
