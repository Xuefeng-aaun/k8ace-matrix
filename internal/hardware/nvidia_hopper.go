package hardware

var NVIDIAMatrixHopper = HardwareMatrix{
	Vendor: VendorNVIDIA,
	Architectures: map[Architecture][]DeviceModel{
		ArchHopper: {
			{
				Name:              "H100_80GB",
				Architecture:      ArchHopper,
				ComputeCapability: "9.0",
				MemoryGB:          80,
				MemoryType:        "HBM3",
				FP32TFLOPS:        51.0,
				DriverMatrix: []DriverVersion{
					{
						Version:           "535.54.03",
						KernelRequirement: ">= 4.15",
						SupportedCards:    []string{"H100"},
						Extensions: map[string]any{
							"branch":        "LTSB",
							"cuda_versions": []string{"12.0", "12.1", "12.2"},
							"notes":         "basic support",
						},
					},
					{
						Version:           "550.54.14",
						KernelRequirement: ">= 4.15",
						SupportedCards:    []string{"H100"},
						Extensions: map[string]any{
							"branch":        "PB",
							"cuda_versions": []string{"12.4", "12.5"},
						},
					},
					{
						Version:           "560.35.03",
						KernelRequirement: ">= 4.15",
						SupportedCards:    []string{"H100"},
						Extensions: map[string]any{
							"recommended": true,
							"branch":        "PB",
							"cuda_versions": []string{"12.6", "12.7"},
							"notes":         "full feature support",
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "nvcr.io/nvidia/k8s-device-plugin:v0.14.0",
					VGPUModes: []VGPUMode{
						{Name: "time_slicing", MinDriver: "450.80.02", Features: []string{"time-slicing"}},
						{Name: "mig", MinDriver: "450.80.02", Features: []string{"isolation"}},
						{Name: "mps", MinDriver: "450.80.02", Features: []string{"mps"}},
					},
				},
				Extensions: map[string]any{
					"model_name":          "NVIDIA H100 80GB",
					"transformer_engine": true,
					"fp8_support":        true,
					"nvlink_version":     4,
					"mig_profiles":       []string{"1g.10gb", "2g.20gb", "3g.40gb", "4g.40gb", "7g.80gb"},
				},
			},
			{
				Name:              "H200_141GB",
				Architecture:      ArchHopper,
				ComputeCapability: "9.0",
				MemoryGB:          141,
				MemoryType:        "HBM3e",
				FP32TFLOPS:        51.0,
				DriverMatrix: []DriverVersion{
					{
						Version:        "550.54.14",
						SupportedCards: []string{"H200"},
						Extensions: map[string]any{
							"cuda_versions": []string{"12.4", "12.5"},
						},
					},
					{
						Version:        "560.35.03",
						SupportedCards: []string{"H200"},
						Extensions: map[string]any{
							"cuda_versions": []string{"12.6", "12.7"},
							"recommended":   true,
						},
					},
				},
				Extensions: map[string]any{
					"model_name": "NVIDIA H200 141GB",
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchHopper: {
			Description:    "NVIDIA Hopper GPU",
			SupportedCards: []string{"H100", "H200"},
			DriverVersions: []string{"535.54.03", "550.54.14", "560.35.03"},
			DriverRange:    ">= 535.54.03",
			Extensions: map[string]any{
				"cuda_min": "12.0",
				"compute_capability": []string{"9.0"},
			},
		},
	},
}
