package hardware

var NVIDIAMatrixAmpere = HardwareMatrix{
	Vendor: VendorNVIDIA,
	Architectures: map[Architecture][]DeviceModel{
		ArchAmpere: {
			{
				Name:              "A100_40GB",
				Architecture:      ArchAmpere,
				ComputeCapability: "8.0",
				MemoryGB:          40,
				MemoryType:        "HBM2",
				FP32TFLOPS:        19.5,
				DriverMatrix: []DriverVersion{
					{
						Version:           "535.54.03",
						KernelRequirement: ">= 4.15",
						SupportedCards:    []string{"A100 40GB", "A100 80GB", "A100 SXM4", "A100 PCIe"},
						Extensions: map[string]any{
							"branch":        "LTSB",
							"eol_date":      "2026-06",
							"cuda_versions": []string{"12.0", "12.1", "12.2"},
							"status":        "stable",
						},
					},
					{
						Version:           "550.54.14",
						KernelRequirement: ">= 4.15",
						SupportedCards:    []string{"A100 40GB", "A100 80GB", "A100 SXM4", "A100 PCIe"},
						Extensions: map[string]any{
							"branch":        "PB",
							"eol_date":      "2026-02",
							"cuda_versions": []string{"12.4", "12.5"},
							"status":        "current",
						},
						DevicePluginReq: DevicePluginRequirement{
							MinVersion: "0.14.0",
							Features:   []string{"time-slicing", "mig"},
						},
					},
					{
						Version:           "560.35.03",
						KernelRequirement: ">= 4.15",
						SupportedCards:    []string{"A100 40GB", "A100 80GB", "A100 SXM4", "A100 PCIe"},
						Extensions: map[string]any{
							"branch":        "PB",
							"eol_date":      "2026-08",
							"cuda_versions": []string{"12.6", "12.7"},
							"status":        "latest",
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "nvcr.io/nvidia/k8s-device-plugin:v0.14.0",
					VGPUModes: []VGPUMode{
						{Name: "time_slicing", MinDriver: "450.80.02", Features: []string{"time-slicing"}},
						{
							Name:      "mig",
							MinDriver: "450.80.02",
							Features:  []string{"isolation", "memory-isolation", "compute-isolation"},
							Extensions: map[string]any{
								"profiles":      []string{"1g.5gb", "2g.10gb", "3g.20gb", "7g.40gb"},
								"max_instances": 7,
							},
						},
						{Name: "mps", MinDriver: "450.80.02", Features: []string{"mps"}},
					},
				},
				Extensions: map[string]any{
					"model_name":    "NVIDIA A100 40GB",
					"tensor_cores":  true,
					"mig_support":   true,
					"max_instances": 7,
				},
			},
			{
				Name:              "A100_80GB",
				Architecture:      ArchAmpere,
				ComputeCapability: "8.0",
				MemoryGB:          80,
				MemoryType:        "HBM2e",
				FP32TFLOPS:        19.5,
				DriverMatrix: []DriverVersion{
					{
						Version:        "535.54.03",
						SupportedCards: []string{"A100 80GB"},
						Extensions: map[string]any{
							"branch":        "LTSB",
							"cuda_versions": []string{"12.0", "12.1", "12.2"},
						},
					},
					{
						Version:        "550.54.14",
						SupportedCards: []string{"A100 80GB"},
						Extensions: map[string]any{
							"branch":        "PB",
							"cuda_versions": []string{"12.4", "12.5"},
						},
					},
					{
						Version:        "560.35.03",
						SupportedCards: []string{"A100 80GB"},
						Extensions: map[string]any{
							"branch":        "PB",
							"cuda_versions": []string{"12.6", "12.7"},
						},
					},
				},
				Extensions: map[string]any{
					"model_name":    "NVIDIA A100 80GB",
					"tensor_cores":  true,
					"mig_support":   true,
					"max_instances": 7,
				},
			},
			{
				Name:              "A10",
				Architecture:      ArchAmpere,
				ComputeCapability: "8.6",
				Extensions: map[string]any{
					"model_name":   "NVIDIA A10",
					"vgpu_license": "nvidia-vpc",
					"mig_support":  false,
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchAmpere: {
			Description:    "NVIDIA Ampere GPU",
			SupportedCards: []string{"A100 40GB", "A100 80GB", "A10"},
			DriverVersions: []string{"535.54.03", "550.54.14", "560.35.03"},
			DriverRange:    ">= 535.54.03",
			Extensions: map[string]any{
				"cuda_min":           "11.0",
				"compute_capability": []string{"8.0", "8.6", "8.7", "8.9"},
			},
		},
	},
}
