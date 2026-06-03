package pipeline

import (
	"fmt"
	"strings"

	"k8ace-matrix/internal/matrix"
)

type BuildUnit struct {
	Hardware        string
	AppName         string
	AppVersion      string
	VariantName     string
	Runtime         string
	Accelerator     string
	BaseRef         string
	BaseVariant     matrix.BaseVariant
	BaseSourceImage string

	BaseImageDest string
	AppImageDest  string

	BaseBuildArgs      map[string]string
	BuildArgs          map[string]string
	AdditionalPackages []string

	// 结构化 Dockerfile 字段
	SystemPackages   []string
	GitRepo          string
	GitRef           string
	AppRoot          string
	Venv             bool
	RequirementsFile string
	Entrypoint       string
	Ports            []string
	Env              map[string]string
	Volumes          []string
}

type AppSpec struct {
	Name      string
	Version   string
	Hardware  string
	FullImage string
}

func DeriveUnits(m *matrix.Matrix, sel Selection) ([]BuildUnit, error) {
	hws := sel.Hardwares
	if len(hws) == 0 {
		return nil, fmt.Errorf("no hardware selected")
	}

	appName := strings.TrimSpace(sel.AppName)
	appVersion := strings.TrimSpace(sel.AppVersion)
	variantFilter := strings.TrimSpace(sel.Variant)

	var appSpecs []AppSpec
	if appName != "" {
		appSpecs = append(appSpecs, AppSpec{Name: appName, Version: appVersion})
	} else if len(sel.Apps) > 0 {
		for _, a := range sel.Apps {
			n, v := splitNameVersion(a)
			appSpecs = append(appSpecs, AppSpec{Name: n, Version: v})
		}
	} else if strings.TrimSpace(sel.PriorityTier) != "" {
		appSpecs = parsePriorityApps(m, sel.PriorityTier)
	}

	if len(appSpecs) == 0 {
		return nil, fmt.Errorf("no app selected")
	}

	regPrefix := strings.TrimSpace(sel.RegistryPrefix)
	if regPrefix == "" {
		regPrefix = strings.TrimSpace(m.RegistryPrefix)
	}
	if regPrefix == "" {
		regPrefix = "k8ace"
	}

	versionSuffix := strings.TrimSpace(sel.VersionSuffix)
	if versionSuffix == "" {
		versionSuffix = "dev"
	}

	var units []BuildUnit
	for _, hw := range hws {
		for _, spec := range appSpecs {
			if strings.TrimSpace(spec.Hardware) != "" && !strings.EqualFold(strings.TrimSpace(spec.Hardware), strings.TrimSpace(hw)) {
				continue
			}

			appDef, ok := findApp(m, spec.Name)
			if !ok {
				return nil, fmt.Errorf("app not found in application_matrix: %s", spec.Name)
			}

			versions := specVersions(appDef, spec.Version)
			for _, ver := range versions {
				for _, v := range appDef.Variants {
					if strings.TrimSpace(v.AppVersion) != "" && !strings.EqualFold(strings.TrimSpace(v.AppVersion), strings.TrimSpace(ver)) {
						continue
					}
					if !containsFold(v.Hardware, hw) {
						continue
					}
					if variantFilter != "" && !strings.EqualFold(variantFilter, v.Name) {
						continue
					}

					baseDef, ok := m.BaseImageMatrix[v.BaseRef]
					if !ok {
						return nil, fmt.Errorf("base_ref not found in base_image_matrix: %s", v.BaseRef)
					}
					baseVar, ok := pickBaseVariant(baseDef.Variants, hw)
					if !ok {
						return nil, fmt.Errorf("no base variant for %s", v.BaseRef)
					}

					baseSource := buildBaseSourceImage(baseDef.Source, baseVar)
					baseDest := fmt.Sprintf("%s/%s-%s", regPrefix, v.BaseRef, baseVar.TagSuffix)

					appDest := spec.FullImage
					if strings.TrimSpace(appDest) == "" {
						appDest = fmt.Sprintf("%s/%s%s-%s-%s-%s", regPrefix, spec.Name, ver, hw, sanitizeName(v.Name), versionSuffix)
					}

					baseBuildArgs := map[string]string{}
					for k, val := range m.BuildArgsOverride[hw] {
						baseBuildArgs[k] = val
					}

					buildArgs := map[string]string{}
					for k, val := range baseBuildArgs {
						buildArgs[k] = val
					}
					for k, val := range v.BuildArgs {
						resolved, err := resolvePlaceholders(val, ver, baseVar)
						if err != nil {
							return nil, err
						}
						buildArgs[k] = resolved
					}

					units = append(units, BuildUnit{
						Hardware:           hw,
						AppName:            spec.Name,
						AppVersion:         ver,
						VariantName:        v.Name,
						Runtime:            v.Runtime,
						Accelerator:        v.Accelerator,
						BaseRef:            v.BaseRef,
						BaseVariant:        baseVar,
						BaseSourceImage:    baseSource,
						BaseImageDest:      baseDest,
						AppImageDest:       appDest,
						BaseBuildArgs:      baseBuildArgs,
						BuildArgs:          buildArgs,
						AdditionalPackages: v.AdditionalPackages,
						SystemPackages:     v.SystemPackages,
						GitRepo:            v.GitRepo,
						GitRef:             v.GitRef,
						AppRoot:            v.AppRoot,
						Venv:               v.Venv,
						RequirementsFile:   v.RequirementsFile,
						Entrypoint:         v.Entrypoint,
						Ports:              v.Ports,
						Env:                v.Env,
						Volumes:            v.Volumes,
					})
				}
			}
		}
	}

	if len(units) == 0 {
		return nil, fmt.Errorf("no build units derived")
	}
	return units, nil
}

func findApp(m *matrix.Matrix, appName string) (matrix.ApplicationDef, bool) {
	for _, apps := range m.ApplicationMatrix {
		for name, def := range apps {
			if strings.EqualFold(name, appName) {
				return def, true
			}
		}
	}
	return matrix.ApplicationDef{}, false
}

func specVersions(def matrix.ApplicationDef, requested string) []string {
	if strings.TrimSpace(requested) != "" {
		return []string{strings.TrimSpace(requested)}
	}
	if len(def.Versions) == 0 {
		return []string{"latest"}
	}
	return []string{def.Versions[0]}
}

func pickBaseVariant(vars []matrix.BaseVariant, hw string) (matrix.BaseVariant, bool) {
	if len(vars) == 0 {
		return matrix.BaseVariant{}, false
	}
	for _, v := range vars {
		if containsFold(v.K8AceCompatible, hw) {
			return v, true
		}
	}
	return vars[0], true
}

func buildBaseSourceImage(source string, baseVar matrix.BaseVariant) string {
	source = strings.TrimSpace(source)
	upstreamOverride := false
	if upstreamSource, ok := baseVar.GetString("upstream_source"); ok && strings.TrimSpace(upstreamSource) != "" {
		source = strings.TrimSpace(upstreamSource)
		upstreamOverride = true
	}
	tagSuffix := strings.TrimSpace(baseVar.TagSuffix)
	if upstreamTag, ok := baseVar.GetString("upstream_tag"); ok && strings.TrimSpace(upstreamTag) != "" {
		tagSuffix = strings.TrimSpace(upstreamTag)
	}
	if source == "" {
		return tagSuffix
	}
	if tagSuffix == "" {
		return source
	}
	if !upstreamOverride && imageRefHasExplicitTag(source) {
		return source
	}
	return source + ":" + tagSuffix
}

func imageRefHasExplicitTag(ref string) bool {
	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	return lastColon > lastSlash
}

func resolvePlaceholders(s, version string, base matrix.BaseVariant) (string, error) {
	out := strings.ReplaceAll(s, "${version}", version)

	for {
		start := strings.Index(out, "${base.")
		if start < 0 {
			break
		}
		end := strings.Index(out[start:], "}")
		if end < 0 {
			break
		}
		end = start + end
		key := strings.TrimSuffix(strings.TrimPrefix(out[start:end+1], "${base."), "}")
		val, ok := base.GetString(key)
		if !ok {
			return "", fmt.Errorf("base placeholder not found: %s", key)
		}
		out = out[:start] + val + out[end+1:]
	}

	return out, nil
}

func containsFold(xs []string, v string) bool {
	for _, x := range xs {
		if strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}

func splitNameVersion(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	i := -1
	for idx, r := range s {
		if r >= '0' && r <= '9' {
			i = idx
			break
		}
	}
	if i <= 0 {
		return s, ""
	}
	name := strings.TrimRight(strings.TrimSpace(s[:i]), "-_")
	ver := strings.TrimLeft(strings.TrimSpace(s[i:]), "-_")
	return name, ver
}

func parsePriorityApps(m *matrix.Matrix, tier string) []AppSpec {
	images := m.PriorityBuildList[tier]
	if len(images) == 0 {
		return nil
	}

	var vendors []string
	for v := range m.BuildArgsOverride {
		vendors = append(vendors, v)
	}

	var out []AppSpec
	for _, img := range images {
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}

		namePart := img
		if i := strings.Index(namePart, "/"); i >= 0 && i < len(namePart)-1 {
			namePart = namePart[i+1:]
		}

		hw := ""
		appWithVer := namePart
		for _, v := range vendors {
			marker := "-" + v + "-"
			if idx := strings.Index(namePart, marker); idx >= 0 {
				hw = v
				appWithVer = namePart[:idx]
				break
			}
		}

		n, ver := splitNameVersion(appWithVer)
		out = append(out, AppSpec{
			Name:      n,
			Version:   ver,
			Hardware:  hw,
			FullImage: img,
		})
	}

	return out
}
