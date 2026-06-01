package matrix

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Matrix struct {
	SchemaVersion     string                       `yaml:"schema_version"`
	Project           string                       `yaml:"project"`
	RegistryPrefix    string                       `yaml:"registry_prefix"`
	BuildPipeline     BuildPipeline                `yaml:"build_pipeline"`
	BaseImageMatrix   map[string]BaseImageDef      `yaml:"base_image_matrix"`
	ApplicationMatrix ApplicationMatrix            `yaml:"application_matrix"`
	BuildArgsOverride map[string]map[string]string `yaml:"build_args_overrides"`
	PriorityBuildList map[string][]string          `yaml:"priority_build_list"`
	NamingConvention  NamingConvention             `yaml:"naming_convention"`
	CICD              CICD                         `yaml:"ci_cd"`
}

type BuildPipeline struct {
	Stages []Stage `yaml:"stages"`
	Cache  Cache   `yaml:"cache"`
}

type Stage struct {
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Parallel        bool     `yaml:"parallel"`
	MatrixDimension string   `yaml:"matrix_dimension"`
	DependsOn       []string `yaml:"depends_on"`
}

type Cache struct {
	Type        string `yaml:"type"`
	TTL         string `yaml:"ttl"`
	KeyTemplate string `yaml:"key_template"`
}

type NamingConvention struct {
	Template string `yaml:"template"`
}

type CICD struct {
	ArgoWorkflows ArgoWorkflows `yaml:"argo_workflows"`
}

type ArgoWorkflows struct {
	Namespace               string           `yaml:"namespace"`
	ServiceAccount          string           `yaml:"service_account"`
	KindDefault             string           `yaml:"kind_default"`
	SubmitModeDefault       string           `yaml:"submit_mode_default"`
	ArgoServer              string           `yaml:"argo_server"`
	KanikoImage             string           `yaml:"kaniko_image"`
	Parallelism             int              `yaml:"parallelism"`
	RegistryMirrors         []string         `yaml:"registry_mirrors"`
	InsecureRegistries      []string         `yaml:"insecure_registries"`
	SkipPushPermissionCheck bool             `yaml:"skip_push_permission_check"`
	BuildContext            ArgoBuildContext `yaml:"build_context"`
	RegistrySecret          string           `yaml:"registry_secret_name"`
	Cache                   ArgoCacheConfig  `yaml:"cache"`
}

type ArgoCacheConfig struct {
	Enabled      bool   `yaml:"enabled"`
	RepoTemplate string `yaml:"repo_template"`
}

type ArgoBuildContext struct {
	Default    string            `yaml:"default"`
	Env        map[string]string `yaml:"env"`
	SecretName string            `yaml:"secret_name"`
	SecretEnv  map[string]string `yaml:"secret_env"`
}

type BaseImageDef struct {
	Source   string        `yaml:"source"`
	Variants []BaseVariant `yaml:"variants"`
}

type BaseVariant struct {
	TagSuffix       string         `yaml:"tag_suffix"`
	K8AceCompatible []string       `yaml:"k8ace_compatible"`
	Extra           map[string]any `yaml:"-"`
}

func (v *BaseVariant) UnmarshalYAML(node *yaml.Node) error {
	var m map[string]any
	if err := node.Decode(&m); err != nil {
		return err
	}
	if s, ok := m["tag_suffix"].(string); ok {
		v.TagSuffix = s
	}
	if xs, ok := m["k8ace_compatible"].([]any); ok {
		var out []string
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		v.K8AceCompatible = out
	}
	delete(m, "tag_suffix")
	delete(m, "k8ace_compatible")
	v.Extra = m
	return nil
}

func (v BaseVariant) GetString(key string) (string, bool) {
	if v.Extra == nil {
		return "", false
	}
	val, ok := v.Extra[key]
	if !ok || val == nil {
		return "", false
	}
	switch t := val.(type) {
	case string:
		return t, true
	default:
		return fmt.Sprint(t), true
	}
}

type ApplicationMatrix map[string]map[string]ApplicationDef

type ApplicationDef struct {
	Versions []string     `yaml:"versions"`
	Variants []AppVariant `yaml:"variants"`
}

type AppVariant struct {
	Name               string            `yaml:"name"`
	BaseRef            string            `yaml:"base_ref"`
	Hardware           []string          `yaml:"hardware"`
	AdditionalPackages []string          `yaml:"additional_packages"`
	BuildArgs          map[string]string `yaml:"build_args"`

	// 结构化 Dockerfile 字段（可选）
	SystemPackages   []string          `yaml:"system_packages"`
	GitRepo          string            `yaml:"git_repo"`
	GitRef           string            `yaml:"git_ref"`
	AppRoot          string            `yaml:"app_root"`
	Venv             bool              `yaml:"venv"`
	RequirementsFile string            `yaml:"requirements_file"`
	Entrypoint       string            `yaml:"entrypoint"`
	Ports            []string          `yaml:"ports"`
	Env              map[string]string `yaml:"env"`
	Volumes          []string          `yaml:"volumes"`

	Extra map[string]any `yaml:"-"`
}

func (v *AppVariant) UnmarshalYAML(node *yaml.Node) error {
	var m map[string]any
	if err := node.Decode(&m); err != nil {
		return err
	}

	if s, ok := m["name"].(string); ok {
		v.Name = s
	}
	if s, ok := m["base_ref"].(string); ok {
		v.BaseRef = s
	}

	if xs, ok := m["hardware"].([]any); ok {
		var out []string
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		v.Hardware = out
	}

	if xs, ok := m["additional_packages"].([]any); ok {
		var out []string
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		v.AdditionalPackages = out
	}

	if mm, ok := m["build_args"].(map[string]any); ok {
		out := map[string]string{}
		for k, val := range mm {
			out[k] = fmt.Sprint(val)
		}
		v.BuildArgs = out
	}

	if xs, ok := m["system_packages"].([]any); ok {
		var out []string
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		v.SystemPackages = out
	}
	if s, ok := m["git_repo"].(string); ok {
		v.GitRepo = s
	}
	if s, ok := m["git_ref"].(string); ok {
		v.GitRef = s
	}
	if s, ok := m["app_root"].(string); ok {
		v.AppRoot = s
	}
	if b, ok := m["venv"].(bool); ok {
		v.Venv = b
	}
	if s, ok := m["requirements_file"].(string); ok {
		v.RequirementsFile = s
	}
	if s, ok := m["entrypoint"].(string); ok {
		v.Entrypoint = s
	}
	if xs, ok := m["ports"].([]any); ok {
		var out []string
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		v.Ports = out
	}
	if mm, ok := m["env"].(map[string]any); ok {
		out := map[string]string{}
		for k, val := range mm {
			out[k] = fmt.Sprint(val)
		}
		v.Env = out
	}
	if xs, ok := m["volumes"].([]any); ok {
		var out []string
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		v.Volumes = out
	}

	delete(m, "name")
	delete(m, "base_ref")
	delete(m, "hardware")
	delete(m, "additional_packages")
	delete(m, "build_args")
	delete(m, "system_packages")
	delete(m, "git_repo")
	delete(m, "git_ref")
	delete(m, "app_root")
	delete(m, "venv")
	delete(m, "requirements_file")
	delete(m, "entrypoint")
	delete(m, "ports")
	delete(m, "env")
	delete(m, "volumes")
	v.Extra = m
	return nil
}
