package hardware

var CambriconMatrixMLU = HardwareMatrix{
	Vendor: VendorCambricon,
	Architectures: map[Architecture][]DeviceModel{
		ArchMLU: {
			{
				Name:         "MLU370",
				Architecture: ArchMLU,
				DriverMatrix: []DriverVersion{
					{
						Version:           "4.20.18",
						KernelRequirement: ">= 3.10",
						SupportedCards:    []string{"MLU370-X8", "MLU370-X4", "MLU590"},
						Extensions: map[string]any{
							"cntoolkit_version": "3.8.2",
							"cnnl_version":      "1.23.0",
							"recommended":       true,
						},
					},
					{
						Version:        "5.10.22",
						SupportedCards: []string{"MLU590", "MLU580"},
						Extensions: map[string]any{
							"cntoolkit_version": "4.0.0",
							"cnnl_version":      "1.25.0",
						},
					},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "k8ace-device-plugin-cambricon",
					Extensions: map[string]any{
						"vgpu_mode": "mlu-share",
					},
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchMLU: {
			Description:    "Cambricon MLU",
			SupportedCards: []string{"MLU370-X8", "MLU370-X4", "MLU580", "MLU590"},
			DriverVersions: []string{"4.20.18", "5.10.22"},
			DriverRange:    ">= 4.20.18",
		},
	},
}

