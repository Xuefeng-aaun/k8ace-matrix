package pipeline

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"k8ace-matrix/internal/matrix"
)

type Selection struct {
	Hardwares    []string
	Apps         []string
	AppName      string
	AppVersion   string
	Variant      string
	Stages       []string
	PriorityTier string

	RegistryPrefix string
	VersionSuffix  string
	Builder        string

	ContextDir  string
	Dockerfile  string
	KanikoImage string
}

type Plan struct {
	Name      string
	Tasks     []Task
	Artifacts Artifacts
}

type Artifacts struct {
	Kind   string
	OutDir string
	Output string
}

type Task struct {
	Name      string
	Stage     string
	Hardware  string
	App       string
	DependsOn []string

	Kaniko KanikoSpec
}

type KanikoSpec struct {
	Image       string
	ContextDir  string
	Dockerfile  string
	Destination string
	BuildArgs   map[string]string
	NoPush      bool
	Cache       CacheSpec
}

type CacheSpec struct {
	Enabled     bool
	Repo        string
	TTL         string
	KeyTemplate string
}

func BuildPlan(m *matrix.Matrix, sel Selection) (*Plan, error) {
	plans, err := BuildPlans(m, sel)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, p := range plans {
		tasks = append(tasks, p.Tasks...)
	}

	return &Plan{
		Name:  planName(sel),
		Tasks: tasks,
	}, nil
}

func BuildPlans(m *matrix.Matrix, sel Selection) ([]*Plan, error) {
	if sel.PriorityTier != "" && (len(sel.Hardwares) == 0) {
		hws := parsePriorityHardwares(m, sel.PriorityTier)
		if len(sel.Hardwares) == 0 {
			sel.Hardwares = hws
		}
	}

	units, err := DeriveUnits(m, sel)
	if err != nil {
		return nil, err
	}

	orderedStages, err := orderedStages(m, sel.Stages)
	if err != nil {
		return nil, err
	}

	regPrefix := strings.TrimSpace(sel.RegistryPrefix)
	if regPrefix == "" {
		regPrefix = strings.TrimSpace(m.RegistryPrefix)
	}
	if regPrefix == "" {
		regPrefix = "k8ace"
	}

	cacheEnabled := strings.EqualFold(m.BuildPipeline.Cache.Type, "registry")
	cacheRepoTpl := strings.TrimSpace(m.CICD.ArgoWorkflows.Cache.RepoTemplate)
	if cacheRepoTpl == "" {
		cacheRepoTpl = "{{ .RegistryPrefix }}/cache/{{ .Hardware }}/{{ .Stage }}"
	}

	contextDir := sel.ContextDir
	if contextDir == "" {
		contextDir = "."
	}

	var plans []*Plan
	for _, u := range units {
		var tasks []Task

		baseTaskName := sanitizeName(strings.Join([]string{"base-image", u.Hardware, u.BaseRef, u.BaseVariant.TagSuffix}, "-"))
		baseDockerfile := fmt.Sprintf("dockerfiles/base_image/%s/%s/%s/Dockerfile", u.Hardware, u.BaseRef, u.BaseVariant.TagSuffix)
		baseArgs := map[string]string{
			"BASE_IMAGE": u.BaseSourceImage,
		}

		cacheRepo := applyRepoTemplate(cacheRepoTpl, regPrefix, u.Hardware, "base_image")

		tasks = append(tasks, Task{
			Name:     baseTaskName,
			Stage:    "base_image",
			Hardware: u.Hardware,
			App:      fmt.Sprintf("%s%s-%s", u.AppName, u.AppVersion, u.VariantName),
			Kaniko: KanikoSpec{
				Image:       sel.KanikoImage,
				ContextDir:  contextDir,
				Dockerfile:  baseDockerfile,
				Destination: u.BaseImageDest,
				BuildArgs:   baseArgs,
				NoPush:      false,
				Cache: CacheSpec{
					Enabled:     cacheEnabled,
					Repo:        cacheRepo,
					TTL:         m.BuildPipeline.Cache.TTL,
					KeyTemplate: m.BuildPipeline.Cache.KeyTemplate,
				},
			},
		})

		appTaskName := sanitizeName(strings.Join([]string{"app-image", u.Hardware, u.AppName, u.AppVersion, u.VariantName}, "-"))
		appDockerfile := fmt.Sprintf("dockerfiles/app_image/%s/%s/%s/%s/Dockerfile", u.Hardware, u.AppName, u.AppVersion, sanitizeName(u.VariantName))
		appArgs := map[string]string{}
		for k, v := range u.BuildArgs {
			appArgs[k] = v
		}
		appArgs["BASE_IMAGE"] = u.BaseImageDest

		cacheRepo = applyRepoTemplate(cacheRepoTpl, regPrefix, u.Hardware, "app_image")

		tasks = append(tasks, Task{
			Name:      appTaskName,
			Stage:     "app_image",
			Hardware:  u.Hardware,
			App:       fmt.Sprintf("%s%s-%s", u.AppName, u.AppVersion, u.VariantName),
			DependsOn: []string{baseTaskName},
			Kaniko: KanikoSpec{
				Image:       sel.KanikoImage,
				ContextDir:  contextDir,
				Dockerfile:  appDockerfile,
				Destination: u.AppImageDest,
				BuildArgs:   appArgs,
				NoPush:      false,
				Cache: CacheSpec{
					Enabled:     cacheEnabled,
					Repo:        cacheRepo,
					TTL:         m.BuildPipeline.Cache.TTL,
					KeyTemplate: m.BuildPipeline.Cache.KeyTemplate,
				},
			},
		})

		prev := appTaskName
		for _, st := range orderedStages {
			if st.Name == "base_image" || st.Name == "app_image" {
				continue
			}
			taskName := sanitizeName(strings.Join([]string{st.Name, u.Hardware, u.AppName, u.AppVersion, u.VariantName}, "-"))
			dockerfile := fmt.Sprintf("dockerfiles/%s/noop/Dockerfile", st.Name)
			cacheRepo = applyRepoTemplate(cacheRepoTpl, regPrefix, u.Hardware, st.Name)

			tasks = append(tasks, Task{
				Name:      taskName,
				Stage:     st.Name,
				Hardware:  u.Hardware,
				App:       fmt.Sprintf("%s%s-%s", u.AppName, u.AppVersion, u.VariantName),
				DependsOn: []string{prev},
				Kaniko: KanikoSpec{
					Image:       sel.KanikoImage,
					ContextDir:  contextDir,
					Dockerfile:  dockerfile,
					Destination: fmt.Sprintf("%s/noop-%s", regPrefix, taskName),
					BuildArgs:   map[string]string{"BASE_IMAGE": "alpine:3.20"},
					NoPush:      true,
					Cache: CacheSpec{
						Enabled:     cacheEnabled,
						Repo:        cacheRepo,
						TTL:         m.BuildPipeline.Cache.TTL,
						KeyTemplate: m.BuildPipeline.Cache.KeyTemplate,
					},
				},
			})
			prev = taskName
		}

		plans = append(plans, &Plan{
			Name:  sanitizeName(strings.Join([]string{"k8ace-matrix", u.Hardware, u.AppName, u.AppVersion, u.VariantName}, "-")),
			Tasks: tasks,
		})
	}

	return plans, nil
}

func parsePriorityHardwares(m *matrix.Matrix, tier string) []string {
	images := m.PriorityBuildList[tier]
	if len(images) == 0 {
		return nil
	}

	vendors := make([]string, 0, len(m.BuildArgsOverride))
	for v := range m.BuildArgsOverride {
		vendors = append(vendors, v)
	}
	sort.Strings(vendors)

	hwSet := map[string]bool{}
	for _, img := range images {
		name := img
		if i := strings.Index(name, "/"); i >= 0 && i < len(name)-1 {
			name = name[i+1:]
		}
		for _, v := range vendors {
			marker := "-" + v + "-"
			if strings.Contains(name, marker) {
				hwSet[v] = true
				break
			}
		}
	}

	var hws []string
	for hw := range hwSet {
		hws = append(hws, hw)
	}
	sort.Strings(hws)
	return hws
}

func planName(sel Selection) string {
	base := "k8ace-matrix"
	if len(sel.Hardwares) == 1 && sel.Hardwares[0] != "" {
		base = sanitizeName(base + "-" + sel.Hardwares[0])
	}
	if len(sel.Apps) == 1 && sel.Apps[0] != "" {
		base = sanitizeName(base + "-" + sel.Apps[0])
	}
	return base
}

func orderedStages(m *matrix.Matrix, requested []string) ([]matrix.Stage, error) {
	if len(m.BuildPipeline.Stages) == 0 {
		return nil, fmt.Errorf("matrix.build_pipeline.stages is empty")
	}

	wantAll := len(requested) == 0 || contains(requested, "all")
	want := map[string]bool{}
	for _, s := range requested {
		want[strings.TrimSpace(s)] = true
	}

	var stages []matrix.Stage
	for _, st := range m.BuildPipeline.Stages {
		if wantAll || want[st.Name] {
			stages = append(stages, st)
		}
	}
	if len(stages) == 0 {
		return nil, fmt.Errorf("no stages selected")
	}

	index := map[string]int{}
	for i, st := range m.BuildPipeline.Stages {
		index[st.Name] = i
	}
	sort.SliceStable(stages, func(i, j int) bool {
		return index[stages[i].Name] < index[stages[j].Name]
	})

	return stages, nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if strings.EqualFold(strings.TrimSpace(x), v) {
			return true
		}
	}
	return false
}

var argoNameRe = regexp.MustCompile(`[^a-z0-9\-]+`)

func sanitizeName(s string) string {
	s = strings.ToLower(s)
	s = argoNameRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if s == "" {
		return "task"
	}
	return s
}

func applyRepoTemplate(tpl, registryPrefix, hardware, stage string) string {
	out := tpl
	out = strings.ReplaceAll(out, "{{ .RegistryPrefix }}", registryPrefix)
	out = strings.ReplaceAll(out, "{{.RegistryPrefix}}", registryPrefix)
	out = strings.ReplaceAll(out, "{{ .Hardware }}", hardware)
	out = strings.ReplaceAll(out, "{{.Hardware}}", hardware)
	out = strings.ReplaceAll(out, "{{ .Stage }}", stage)
	out = strings.ReplaceAll(out, "{{.Stage}}", stage)
	return out
}
