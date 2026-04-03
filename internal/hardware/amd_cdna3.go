package hardware

var AMDMatrixCDNA3 = HardwareMatrix{
	Vendor: VendorAMD,
	Architectures: map[Architecture][]DeviceModel{
		ArchCDNA3: {
			{
				Name:         "MI300X",
				Architecture: ArchCDNA3,
				MemoryGB:     192,
				MemoryType:   "HBM3",
				FP32TFLOPS:   81.7,
				DriverMatrix: []DriverVersion{
					{
						Version:           "6.0.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"MI300X"},
						Extensions: map[string]any{
							"rocm_version": "6.0",
						},
					},
					{
						Version:           "6.3.0",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"MI300X"},
						Extensions: map[string]any{
							"rocm_version": "6.3",
							"recommended":  true,
						},
					},
					{
						Version:        "7.0.0",
						SupportedCards: []string{"MI300X"},
						Extensions: map[string]any{
							"rocm_version": "7.0",
							"status":       "preview",
							"notes":        "MI350 requires ROCm 7.0+",
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "amd.com/gpu",
					Extensions: map[string]any{
						"modes": []string{"whole_card"},
					},
				},
				Extensions: map[string]any{
					"model_name":       "AMD Instinct MI300X",
					"gfx_arch":         "gfx942",
					"unified_memory":   true,
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchCDNA3: {
			Description:    "AMD CDNA3 (ROCm)",
			SupportedCards: []string{"MI300X"},
			DriverVersions: []string{"6.0.0", "6.3.0", "7.0.0"},
			DriverRange:    "ROCm 6.0..6.3 (7.0 preview)",
			Extensions: map[string]any{
				"gfx_arch":  "gfx942",
				"rocm_min":  "6.0",
				"rocm_max":  "6.3",
			},
		},
	},
}

