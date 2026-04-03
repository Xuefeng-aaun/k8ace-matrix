package hardware

func mergeMatrices(vendor Vendor, mats ...HardwareMatrix) HardwareMatrix {
	out := HardwareMatrix{
		Vendor:           vendor,
		Architectures:    map[Architecture][]DeviceModel{},
		ArchitectureInfo: map[Architecture]ArchitectureInfo{},
	}

	for _, m := range mats {
		for arch, models := range m.Architectures {
			out.Architectures[arch] = append(out.Architectures[arch], models...)
		}
		for arch, info := range m.ArchitectureInfo {
			out.ArchitectureInfo[arch] = info
		}
	}

	if len(out.Architectures) == 0 {
		out.Architectures = nil
	}
	if len(out.ArchitectureInfo) == 0 {
		out.ArchitectureInfo = nil
	}

	return out
}

var (
	NVIDIAMatrix    = mergeMatrices(VendorNVIDIA, NVIDIAMatrixAmpere, NVIDIAMatrixHopper)
	AMDMatrix       = mergeMatrices(VendorAMD, AMDMatrixCDNA2, AMDMatrixCDNA3)
	IntelMatrix     = mergeMatrices(VendorIntel, IntelMatrixXeHPG)
	HygonMatrix     = mergeMatrices(VendorHygon, HygonMatrixDCUGen1, HygonMatrixDCUGen2)
	AscendMatrix    = mergeMatrices(VendorHuawei, AscendMatrixAscend)
	CambriconMatrix = mergeMatrices(VendorCambricon, CambriconMatrixMLU)
	CPUMatrix       = mergeMatrices(VendorCPU, CPUMatrixGeneric)

	AllMatrices = map[Vendor]*HardwareMatrix{
		VendorNVIDIA:    &NVIDIAMatrix,
		VendorAMD:       &AMDMatrix,
		VendorIntel:     &IntelMatrix,
		VendorHygon:     &HygonMatrix,
		VendorHuawei:    &AscendMatrix,
		VendorCambricon: &CambriconMatrix,
		VendorCPU:       &CPUMatrix,
	}
)
