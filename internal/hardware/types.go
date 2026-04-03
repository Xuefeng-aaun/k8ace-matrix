package hardware

import (
	"encoding/json"
	"fmt"
)

type Vendor string

const (
	VendorNVIDIA       Vendor = "NVIDIA"
	VendorAMD          Vendor = "AMD"
	VendorIntel        Vendor = "INTEL"
	VendorHygon        Vendor = "HYGON"
	VendorHuawei       Vendor = "HUAWEI"
	VendorCambricon    Vendor = "CAMBRICON"
	VendorMetaX        Vendor = "METAX"
	VendorIluvatar     Vendor = "ILUVATAR"
	VendorEnflame      Vendor = "ENFLAME"
	VendorKunlunxin    Vendor = "KUNLUNXIN"
	VendorMooreThreads Vendor = "MOORETHREADS"
	VendorCPU          Vendor = "CPU"
)

type Architecture string

const (
	ArchAmpere    Architecture = "ampere"
	ArchAda       Architecture = "ada"
	ArchHopper    Architecture = "hopper"
	ArchBlackwell Architecture = "blackwell"
	ArchVolta     Architecture = "volta"
	ArchTuring    Architecture = "turing"
	ArchDCU       Architecture = "dcu"
	ArchDCUGen1   Architecture = "dcu_gen1"
	ArchDCUGen2   Architecture = "dcu_gen2"
	ArchAscend    Architecture = "ascend"
	ArchMLU       Architecture = "mlu"
	ArchCDNA2     Architecture = "cdna2"
	ArchCDNA3     Architecture = "cdna3"
	ArchXeHPG     Architecture = "xehpg"
	ArchXeHPC     Architecture = "xehpc"
	ArchMetaX     Architecture = "metax"
	ArchCoreX     Architecture = "corex"
	ArchGCU       Architecture = "gcu"
	ArchXPU       Architecture = "xpu"
	ArchMUSA      Architecture = "musa"
	ArchGeneric   Architecture = "generic"
)

type DeviceModel struct {
	Name               string       `json:"name"`
	Architecture       Architecture `json:"architecture"`
	ComputeCapability  string       `json:"compute_capability,omitempty"`
	MemoryGB           int          `json:"memory_gb,omitempty"`
	MemoryType         string       `json:"memory_type,omitempty"`
	MemoryBandwidthGBs float64      `json:"memory_bandwidth_gbps,omitempty"`
	FP32TFLOPS         float64      `json:"fp32_tflops,omitempty"`
	FP16TFLOPS         float64      `json:"fp16_tflops,omitempty"`
	INT8TOPS           float64      `json:"int8_tops,omitempty"`

	DriverMatrix []DriverVersion `json:"driver_matrix,omitempty"`
	K8sSupport   K8sConfig       `json:"k8s_support,omitempty"`

	Extensions map[string]any `json:"extensions,omitempty"`
}

type DriverVersion struct {
	Version           string   `json:"version"`
	ReleaseDate       string   `json:"release_date,omitempty"`
	CUDAVersion       string   `json:"cuda_version,omitempty"`
	ROCmVersion       string   `json:"rocm_version,omitempty"`
	LevelZeroVersion  string   `json:"level_zero_version,omitempty"`
	KernelRequirement string   `json:"kernel_requirement,omitempty"`
	SupportedCards    []string `json:"supported_cards,omitempty"`

	DevicePluginReq DevicePluginRequirement `json:"device_plugin_req,omitempty"`
	Extensions      map[string]any          `json:"extensions,omitempty"`
}

type DevicePluginRequirement struct {
	MinVersion string         `json:"min_version,omitempty"`
	Features   []string       `json:"features,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

type K8sConfig struct {
	DefaultDevicePlugin string         `json:"default_device_plugin,omitempty"`
	VGPUModes           []VGPUMode     `json:"vgpu_modes,omitempty"`
	Extensions          map[string]any `json:"extensions,omitempty"`
}

type VGPUMode struct {
	Name       string         `json:"name"`
	MinDriver  string         `json:"min_driver,omitempty"`
	Features   []string       `json:"features,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

type HardwareMatrix struct {
	Vendor           Vendor                            `json:"vendor"`
	Architectures    map[Architecture][]DeviceModel    `json:"architectures"`
	ArchitectureInfo map[Architecture]ArchitectureInfo `json:"architecture_info,omitempty"`
	Extensions       map[string]any                    `json:"extensions,omitempty"`
}

type ArchitectureInfo struct {
	Description    string         `json:"description,omitempty"`
	SupportedCards []string       `json:"supported_cards,omitempty"`
	DriverVersions []string       `json:"driver_versions,omitempty"`
	DriverRange    string         `json:"driver_range,omitempty"`
	Extensions     map[string]any `json:"extensions,omitempty"`
}

func (m *HardwareMatrix) GetDeviceModel(name string) (*DeviceModel, error) {
	if m == nil {
		return nil, fmt.Errorf("nil matrix")
	}
	for _, models := range m.Architectures {
		for _, model := range models {
			if model.Name == name {
				cp := deepCopyModel(&model)
				return cp, nil
			}
		}
	}
	return nil, fmt.Errorf("device model %s not found for vendor %s", name, m.Vendor)
}

func (dm *DeviceModel) GetBestDriverVersion() *DriverVersion {
	if dm == nil || len(dm.DriverMatrix) == 0 {
		return nil
	}
	best := &dm.DriverMatrix[len(dm.DriverMatrix)-1]
	for i := range dm.DriverMatrix {
		drv := &dm.DriverMatrix[i]
		if drv.Extensions != nil {
			if v, ok := drv.Extensions["recommended"].(bool); ok && v {
				return drv
			}
		}
	}
	return best
}

func (dm *DeviceModel) ToJSON() ([]byte, error) {
	if dm == nil {
		return nil, fmt.Errorf("nil device model")
	}
	return json.MarshalIndent(dm, "", "  ")
}

func (m *HardwareMatrix) RegisterDeviceModel(model DeviceModel) error {
	if m == nil {
		return fmt.Errorf("nil matrix")
	}
	if m.Architectures == nil {
		m.Architectures = map[Architecture][]DeviceModel{}
	}
	m.Architectures[model.Architecture] = append(m.Architectures[model.Architecture], model)
	return nil
}

func deepCopyModel(src *DeviceModel) *DeviceModel {
	dst := *src

	if src.Extensions != nil {
		dst.Extensions = map[string]any{}
		for k, v := range src.Extensions {
			dst.Extensions[k] = v
		}
	}

	if len(src.DriverMatrix) > 0 {
		dst.DriverMatrix = make([]DriverVersion, len(src.DriverMatrix))
		for i := range src.DriverMatrix {
			dst.DriverMatrix[i] = src.DriverMatrix[i]
			if src.DriverMatrix[i].Extensions != nil {
				dst.DriverMatrix[i].Extensions = map[string]any{}
				for k, v := range src.DriverMatrix[i].Extensions {
					dst.DriverMatrix[i].Extensions[k] = v
				}
			}
			if src.DriverMatrix[i].DevicePluginReq.Extensions != nil {
				dst.DriverMatrix[i].DevicePluginReq.Extensions = map[string]any{}
				for k, v := range src.DriverMatrix[i].DevicePluginReq.Extensions {
					dst.DriverMatrix[i].DevicePluginReq.Extensions[k] = v
				}
			}
		}
	}

	if len(src.K8sSupport.VGPUModes) > 0 {
		dst.K8sSupport.VGPUModes = make([]VGPUMode, len(src.K8sSupport.VGPUModes))
		for i := range src.K8sSupport.VGPUModes {
			dst.K8sSupport.VGPUModes[i] = src.K8sSupport.VGPUModes[i]
			if src.K8sSupport.VGPUModes[i].Extensions != nil {
				dst.K8sSupport.VGPUModes[i].Extensions = map[string]any{}
				for k, v := range src.K8sSupport.VGPUModes[i].Extensions {
					dst.K8sSupport.VGPUModes[i].Extensions[k] = v
				}
			}
		}
	}

	if src.K8sSupport.Extensions != nil {
		dst.K8sSupport.Extensions = map[string]any{}
		for k, v := range src.K8sSupport.Extensions {
			dst.K8sSupport.Extensions[k] = v
		}
	}

	return &dst
}
