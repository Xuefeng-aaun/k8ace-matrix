package hardware

var HygonMatrixDCUGen2 = HardwareMatrix{
	Vendor: VendorHygon,
	Architectures: map[Architecture][]DeviceModel{
		ArchDCUGen2: {
			{
				Name:         "Z100L",
				Architecture: ArchDCUGen2,
				MemoryGB:     64,
				MemoryType:   "HBM2e",
				FP32TFLOPS:   18.5,
				DriverMatrix: []DriverVersion{
					{
						Version:           "DTK-24.04.3",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"Z100L", "K200", "Z200"},
						Extensions: map[string]any{
							"dtk_version":     "24.04.3",
							"rocm_version":    "6.0.0",
							"hy_smi_version":  "1.6.0",
							"format":          "modern",
							"status":          "recommended",
							"critical":        true,
							"recommended":     true,
							"notes":           "K8s share requires DTK >= 24.04",
						},
					},
					{
						Version:           "DTK-24.06",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"Z100L", "K200", "Z200"},
						Extensions: map[string]any{
							"dtk_version":     "24.06",
							"rocm_version":    "6.1.0",
							"hy_smi_version":  "1.7.0",
							"status":          "latest",
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "k8ace-device-plugin-hygon",
					VGPUModes: []VGPUMode{
						{
							Name:      "dcu_share",
							MinDriver: "DTK-24.04",
							Features:  []string{"memory-limit", "compute-limit"},
							Extensions: map[string]any{
								"max_vgpus_per_gpu": 8,
							},
						},
					},
					Extensions: map[string]any{
						"modes": []string{"whole_card", "dcu_share"},
						"resources": []string{"hygon.com/dcunum", "hygon.com/dcumem", "hygon.com/dcucores"},
					},
				},
				Extensions: map[string]any{
					"model_name":              "海光 DCU Z100L (深算二号)",
					"dcu_generation":          2,
					"rocm_compat":             "6.0",
					"vgpu_capable":            true,
					"deepseek_r1_support":     true,
					"ai_inference_optimized":  true,
				},
			},
			{
				Name:         "K200",
				Architecture: ArchDCUGen2,
				MemoryGB:     64,
				MemoryType:   "HBM2e",
				FP32TFLOPS:   18.5,
				DriverMatrix: []DriverVersion{
					{
						Version:        "DTK-24.04.3",
						SupportedCards: []string{"K200"},
						Extensions: map[string]any{
							"dtk_version":    "24.04.3",
							"recommended":    true,
						},
					},
				},
				Extensions: map[string]any{
					"model_name":     "海光 DCU K200 (深算二号)",
					"dcu_generation": 2,
					"vgpu_capable":   true,
				},
			},
			{
				Name:         "Z200",
				Architecture: ArchDCUGen2,
				MemoryGB:     64,
				MemoryType:   "HBM2e",
				DriverMatrix: []DriverVersion{
					{
						Version:        "DTK-24.04.3",
						SupportedCards: []string{"Z200"},
						Extensions: map[string]any{
							"dtk_version":  "24.04.3",
							"recommended":  true,
						},
					},
				},
				Extensions: map[string]any{
					"model_name":     "海光 DCU Z200 (深算二号)",
					"dcu_generation": 2,
					"vgpu_capable":   true,
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchDCUGen2: {
			Description:    "HYGON DCU Gen2 (DTK required, sharing supported)",
			SupportedCards: []string{"Z100L", "K200", "Z200"},
			DriverVersions: []string{"DTK-24.04.3", "DTK-24.06"},
			DriverRange:    ">= DTK-24.04",
			Extensions: map[string]any{
				"versioning": "dtk",
				"kernel_min": "5.15",
				"critical_requirements": map[string]any{
					"dtk_version":    ">= 24.04",
					"hy_smi_version": ">= 1.6.0",
				},
			},
		},
	},
}

