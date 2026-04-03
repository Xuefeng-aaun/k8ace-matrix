package hardware

var HygonMatrixDCUGen1 = HardwareMatrix{
	Vendor: VendorHygon,
	Architectures: map[Architecture][]DeviceModel{
		ArchDCUGen1: {
			{
				Name:         "Z100",
				Architecture: ArchDCUGen1,
				MemoryGB:     32,
				MemoryType:   "HBM2",
				FP32TFLOPS:   10.5,
				DriverMatrix: []DriverVersion{
					{
						Version:           "DTK-23.10",
						KernelRequirement: ">= 4.18",
						SupportedCards:    []string{"Z100", "K100"},
						Extensions: map[string]any{
							"dtk_version":     "23.10",
							"rocm_version":    "5.6.0",
							"hy_smi_version":  "1.4.0",
							"format":          "legacy",
							"status":          "legacy",
						},
					},
					{
						Version:           "DTK-24.04",
						KernelRequirement: ">= 5.15",
						SupportedCards:    []string{"Z100", "K100"},
						Extensions: map[string]any{
							"dtk_version":     "24.04",
							"rocm_version":    "6.0.0",
							"hy_smi_version":  "1.6.0",
							"format":          "modern",
							"status":          "current",
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "k8ace-device-plugin-hygon",
					Extensions: map[string]any{
						"modes": []string{"whole_card"},
						"notes": "DTK < 24.04 does not support sharing",
					},
				},
				Extensions: map[string]any{
					"model_name":      "海光 DCU Z100 (深算一号)",
					"dcu_generation":  1,
					"rocm_compat":     "5.6",
					"vgpu_capable":    false,
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchDCUGen1: {
			Description:    "HYGON DCU Gen1 (DTK required)",
			SupportedCards: []string{"Z100", "K100"},
			DriverVersions: []string{"DTK-23.10", "DTK-24.04"},
			DriverRange:    "DTK-23.10..DTK-24.04",
			Extensions: map[string]any{
				"versioning": "dtk",
				"kernel_min": "4.18",
			},
		},
	},
}

