package hardware

var AscendMatrixAscend = HardwareMatrix{
	Vendor: VendorHuawei,
	Architectures: map[Architecture][]DeviceModel{
		ArchAscend: {
			{
				Name:         "Ascend910B",
				Architecture: ArchAscend,
				DriverMatrix: []DriverVersion{
					{
						Version:           "23.0.3",
						KernelRequirement: ">= 4.19",
						SupportedCards:    []string{"Ascend910B", "Ascend910A", "Ascend310P"},
						Extensions: map[string]any{
							"cann_version":     "8.0.RC2",
							"firmware_version": "7.1.0.5.220",
						},
					},
					{
						Version:        "24.1.0",
						SupportedCards: []string{"Ascend910C", "Ascend910B4"},
						Extensions: map[string]any{
							"cann_version":     "8.0.RC3",
							"firmware_version": "7.3.0.1.231",
							"recommended":      true,
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "ascend-device-plugin",
					Extensions: map[string]any{
						"vgpu_mode":    "vNPU",
						"vnpu_support": true,
					},
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchAscend: {
			Description:    "Huawei Ascend NPU",
			SupportedCards: []string{"Ascend910A", "Ascend910B", "Ascend910C", "Ascend310P"},
			DriverVersions: []string{"23.0.3", "24.1.0"},
			DriverRange:    ">= 23.0.3",
		},
	},
}

