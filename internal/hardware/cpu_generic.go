package hardware

var CPUMatrixGeneric = HardwareMatrix{
	Vendor: VendorCPU,
	Architectures: map[Architecture][]DeviceModel{
		ArchGeneric: {
			{
				Name:         "Generic-CPU",
				Architecture: ArchGeneric,
				DriverMatrix: []DriverVersion{
					{Version: "none", KernelRequirement: ">= 3.10", SupportedCards: []string{"any"}},
				},
				K8sSupport: K8sConfig{
					DefaultDevicePlugin: "none",
				},
				Extensions: map[string]any{
					"compute_backend": "OpenBLAS/MKL",
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchGeneric: {
			Description:    "Generic CPU fallback",
			SupportedCards: []string{"any"},
			DriverVersions: []string{"none"},
		},
	},
}

