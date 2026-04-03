package hardware

var AMDMatrixCDNA2 = HardwareMatrix{
	Vendor: VendorAMD,
	Architectures: map[Architecture][]DeviceModel{
		ArchCDNA2: {
			{
				Name:         "MI210",
				Architecture: ArchCDNA2,
				MemoryGB:     64,
				MemoryType:   "HBM2",
				FP32TFLOPS:   22.6,
				DriverMatrix: []DriverVersion{
					{
						Version:           "5.7.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"MI210", "MI250", "MI250X"},
						Extensions: map[string]any{
							"rocm_version": "5.7",
							"status":       "legacy",
						},
					},
					{
						Version:           "6.0.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"MI210", "MI250", "MI250X"},
						Extensions: map[string]any{
							"rocm_version": "6.0",
						},
					},
					{
						Version:           "6.1.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"MI210", "MI250", "MI250X"},
						Extensions: map[string]any{
							"rocm_version": "6.1",
						},
					},
					{
						Version:           "6.2.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"MI210", "MI250", "MI250X"},
						Extensions: map[string]any{
							"rocm_version": "6.2",
						},
					},
					{
						Version:           "6.3.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"MI210", "MI250", "MI250X"},
						Extensions: map[string]any{
							"rocm_version": "6.3",
							"recommended":  true,
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "amd.com/gpu",
					Extensions: map[string]any{
						"modes": []string{"whole_card"},
						"notes": "no vGPU; whole device only",
					},
				},
				Extensions: map[string]any{
					"model_name": "AMD Instinct MI210",
					"gfx_arch":   "gfx90a",
				},
			},
			{
				Name:         "MI250X",
				Architecture: ArchCDNA2,
				MemoryGB:     128,
				MemoryType:   "HBM2e",
				FP32TFLOPS:   47.9,
				DriverMatrix: []DriverVersion{
					{
						Version:        "6.0.0",
						SupportedCards: []string{"MI250X"},
						Extensions: map[string]any{
							"rocm_version": "6.0",
						},
					},
					{
						Version:        "6.3.0",
						SupportedCards: []string{"MI250X"},
						Extensions: map[string]any{
							"rocm_version": "6.3",
							"recommended":  true,
						},
					},
				},
				Extensions: map[string]any{
					"model_name": "AMD Instinct MI250X",
					"gfx_arch":   "gfx90a",
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchCDNA2: {
			Description:    "AMD CDNA2 (ROCm)",
			SupportedCards: []string{"MI210", "MI250X"},
			DriverVersions: []string{"5.7.0", "6.0.0", "6.1.0", "6.2.0", "6.3.0"},
			DriverRange:    "ROCm 5.4..6.3",
			Extensions: map[string]any{
				"gfx_arch":  "gfx90a",
				"rocm_min":  "5.4",
				"rocm_max":  "6.3",
			},
		},
	},
}

