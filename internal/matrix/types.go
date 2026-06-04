package matrix

import (
	"fmt"
	"sort"

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
	Kaniko                  ArgoKanikoConfig `yaml:"kaniko"`
	Parallelism             int              `yaml:"parallelism"`
	RegistryMirrors         []string         `yaml:"registry_mirrors"`
	InsecureRegistries      []string         `yaml:"insecure_registries"`
	SkipPushPermissionCheck bool             `yaml:"skip_push_permission_check"`
	BuildContext            ArgoBuildContext `yaml:"build_context"`
	RegistrySecret          string           `yaml:"registry_secret_name"`
	Cache                   ArgoCacheConfig  `yaml:"cache"`
}

type ArgoKanikoConfig struct {
	SnapshotMode   string `yaml:"snapshot_mode"`
	SingleSnapshot bool   `yaml:"single_snapshot"`
	UseNewRun      bool   `yaml:"use_new_run"`
	Cleanup        bool   `yaml:"cleanup"`
	Reproducible   bool   `yaml:"reproducible"`
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
	Type     string         `yaml:"type"`
	Versions []string       `yaml:"versions"`
	Variants []AppVariant   `yaml:"variants"`
	Extra    map[string]any `yaml:"-"`
}

type AppVariant struct {
	Name               string            `yaml:"name"`
	AppType            string            `yaml:"type"`
	AppVersion         string            `yaml:"app_version"`
	Runtime            string            `yaml:"runtime"`
	Accelerator        string            `yaml:"accelerator"`
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

func (d *ApplicationDef) UnmarshalYAML(node *yaml.Node) error {
	var m map[string]any
	if err := node.Decode(&m); err != nil {
		return err
	}

	d.Type = firstString(m, "type", "app_type")

	if rawVersions, ok := m["versions"]; ok {
		switch versions := rawVersions.(type) {
		case []any:
			for _, x := range versions {
				if s, ok := x.(string); ok {
					d.Versions = append(d.Versions, s)
				}
			}
		case map[string]any:
			d.decodeVersionMap(versions)
		}
	}

	if xs, ok := m["variants"].([]any); ok {
		for _, x := range xs {
			if mm, ok := anyMap(x); ok {
				v := decodeAppVariantMap(mm)
				if v.AppType == "" {
					v.AppType = d.Type
				}
				d.Variants = append(d.Variants, v)
			}
		}
	}

	delete(m, "type")
	delete(m, "app_type")
	delete(m, "versions")
	delete(m, "variants")
	d.Extra = m
	return nil
}

func (d *ApplicationDef) decodeVersionMap(versions map[string]any) {
	versionKeys := sortedKeys(versions)
	for _, version := range versionKeys {
		d.Versions = append(d.Versions, version)
		versionMap, ok := anyMap(versions[version])
		if !ok {
			continue
		}
		runtimes, ok := anyMap(versionMap["runtimes"])
		if !ok {
			continue
		}
		for _, runtime := range sortedKeys(runtimes) {
			accelerators, ok := anyMap(runtimes[runtime])
			if !ok {
				continue
			}
			for _, accelerator := range sortedKeys(accelerators) {
				variantMap, ok := anyMap(accelerators[accelerator])
				if !ok {
					continue
				}
				v := decodeAppVariantMap(variantMap)
				if v.AppType == "" {
					v.AppType = d.Type
				}
				if v.AppVersion == "" {
					v.AppVersion = version
				}
				if v.Runtime == "" {
					v.Runtime = runtime
				}
				if v.Accelerator == "" {
					v.Accelerator = accelerator
				}
				d.Variants = append(d.Variants, v)
			}
		}
	}
}

func (v *AppVariant) UnmarshalYAML(node *yaml.Node) error {
	var m map[string]any
	if err := node.Decode(&m); err != nil {
		return err
	}

	*v = decodeAppVariantMap(m)
	return nil
}

func decodeAppVariantMap(m map[string]any) AppVariant {
	v := AppVariant{}

	v.Name = firstString(m, "name")
	v.AppType = firstString(m, "type", "app_type")
	v.AppVersion = firstString(m, "app_version", "version")
	v.Runtime = firstString(m, "runtime")
	v.Accelerator = firstString(m, "accelerator", "accelerator_version")
	v.BaseRef = firstString(m, "base_ref")
	v.Hardware = stringList(m["hardware"])
	v.AdditionalPackages = stringList(m["additional_packages"])
	v.BuildArgs = stringMap(m["build_args"])
	v.SystemPackages = stringList(m["system_packages"])
	v.GitRepo = firstString(m, "git_repo")
	v.GitRef = firstString(m, "git_ref")
	v.AppRoot = firstString(m, "app_root")
	if b, ok := m["venv"].(bool); ok {
		v.Venv = b
	}
	v.RequirementsFile = firstString(m, "requirements_file")
	v.Entrypoint = firstString(m, "entrypoint")
	v.Ports = stringList(m["ports"])
	v.Env = stringMap(m["env"])
	v.Volumes = stringList(m["volumes"])

	extra := map[string]any{}
	for k, val := range m {
		extra[k] = val
	}
	for _, key := range []string{
		"name", "type", "app_type", "app_version", "version", "runtime", "accelerator", "accelerator_version",
		"base_ref", "hardware", "additional_packages", "build_args", "system_packages", "git_repo", "git_ref",
		"app_root", "venv", "requirements_file", "entrypoint", "ports", "env", "volumes",
	} {
		delete(extra, key)
	}
	v.Extra = extra
	return v
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := m[key].(string); ok {
			return s
		}
	}
	return ""
}

func stringList(raw any) []string {
	xs, ok := raw.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, x := range xs {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func stringMap(raw any) map[string]string {
	m, ok := anyMap(raw)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for k, val := range m {
		out[k] = fmt.Sprint(val)
	}
	return out
}

func anyMap(raw any) (map[string]any, bool) {
	m, ok := raw.(map[string]any)
	return m, ok
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
