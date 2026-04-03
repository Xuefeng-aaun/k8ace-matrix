package hardware

var IntelMatrixXeHPG = HardwareMatrix{
	Vendor: VendorIntel,
	Architectures: map[Architecture][]DeviceModel{
		ArchXeHPG: {
			{
				Name:         "Max_1550",
				Architecture: ArchXeHPG,
				MemoryGB:     128,
				MemoryType:   "HBM2",
				FP32TFLOPS:   52.0,
				DriverMatrix: []DriverVersion{
					{
						Version:           "2024.1.0",
						KernelRequirement: ">= 5.14",
						SupportedCards:    []string{"Intel Data Center GPU Max 1550"},
						Extensions: map[string]any{
							"oneapi_version": "2024.1",
						},
					},
					{
						Version:           "2024.2.0",
						KernelRequirement: ">= 5.14",
						SupportedCards:    []string{"Intel Data Center GPU Max 1550", "Intel Data Center GPU Max 1100"},
						Extensions: map[string]any{
							"oneapi_version": "2024.2",
						},
					},
					{
						Version:           "2025.0.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"Intel Data Center GPU Max 1550", "Intel Data Center GPU Max 1100"},
						Extensions: map[string]any{
							"oneapi_version": "2025.0",
							"recommended":    true,
							"notes":          "vLLM XPU backend recommended",
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "gpu.intel.com/i915",
					Extensions: map[string]any{
						"modes": []string{"whole_card"},
					},
				},
				Extensions: map[string]any{
					"model_name": "Intel Data Center GPU Max 1550",
					"xe_cores":   128,
				},
			},
			{
				Name:         "Max_1100",
				Architecture: ArchXeHPG,
				MemoryGB:     48,
				MemoryType:   "HBM2",
				DriverMatrix: []DriverVersion{
					{
						Version:        "2024.2.0",
						SupportedCards: []string{"Intel Data Center GPU Max 1100"},
						Extensions: map[string]any{
							"oneapi_version": "2024.2",
						},
					},
					{
						Version:        "2025.0.0",
						SupportedCards: []string{"Intel Data Center GPU Max 1100"},
						Extensions: map[string]any{
							"oneapi_version": "2025.0",
							"recommended":    true,
						},
					},
				},
				Extensions: map[string]any{
					"model_name": "Intel Data Center GPU Max 1100",
					"xe_cores":   56,
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchXeHPG: {
			Description:    "Intel Xe HPG (Data Center GPU Max series)",
			SupportedCards: []string{"Intel Data Center GPU Max 1550", "Intel Data Center GPU Max 1100"},
			DriverVersions: []string{"2024.1.0", "2024.2.0", "2025.0.0"},
			DriverRange:    ">= 2024.1.0",
			Extensions: map[string]any{
				"oneapi_min": "2024.0",
				"features":   []string{"xess", "sycl"},
			},
		},
	},
}

