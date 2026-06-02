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

	Kaniko               KanikoSpec
	HostDriver           HostDriverSpec
	TestImage            string
	TestCommands         []string
	TestResourceLimits   map[string]string
	TestResourceRequests map[string]string
}

type HostDriverSpec struct {
	Image            string
	Commands         []string
	ResourceLimits   map[string]string
	ResourceRequests map[string]string
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

	builder := strings.TrimSpace(sel.Builder)
	if builder == "" {
		builder = "kaniko"
	}
	if !strings.EqualFold(builder, "kaniko") {
		return nil, fmt.Errorf("unsupported builder: %s", sel.Builder)
	}

	units, err := DeriveUnits(m, sel)
	if err != nil {
		return nil, err
	}

	resolvedStages, err := resolveStages(m, sel.Stages)
	if err != nil {
		return nil, err
	}

	overrideStageName, overrideDockerfile, err := resolveDockerfileOverride(sel.Stages, sel.Dockerfile)
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

	cacheEnabled := strings.EqualFold(m.BuildPipeline.Cache.Type, "registry") && m.CICD.ArgoWorkflows.Cache.Enabled
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
		stageTaskNames := map[string]string{}
		for _, st := range resolvedStages {
			taskName := taskNameForStage(st.Name, u)
			stageTaskNames[st.Name] = taskName

			task := buildStageTask(
				st.Name,
				u,
				sel.KanikoImage,
				contextDir,
				regPrefix,
				cacheEnabled,
				cacheRepoTpl,
				m.BuildPipeline.Cache,
				overrideStageName,
				overrideDockerfile,
			)
			tasks = append(tasks, task)
		}

		for i := range tasks {
			stageDef, ok := stageByName(resolvedStages, tasks[i].Stage)
			if !ok {
				return nil, fmt.Errorf("stage definition not found: %s", tasks[i].Stage)
			}
			for _, depName := range stageDef.DependsOn {
				taskDepName, ok := stageTaskNames[depName]
				if !ok {
					return nil, fmt.Errorf("resolved dependency %s not found for stage %s", depName, stageDef.Name)
				}
				tasks[i].DependsOn = append(tasks[i].DependsOn, taskDepName)
			}
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

func resolveStages(m *matrix.Matrix, requested []string) ([]matrix.Stage, error) {
	if len(m.BuildPipeline.Stages) == 0 {
		return nil, fmt.Errorf("matrix.build_pipeline.stages is empty")
	}

	stageDefs := map[string]matrix.Stage{}
	for _, st := range m.BuildPipeline.Stages {
		stageDefs[st.Name] = st
	}

	wantAll := len(requested) == 0 || contains(requested, "all")
	required := map[string]bool{}

	var collectDeps func(string, map[string]bool) error
	collectDeps = func(stageName string, visiting map[string]bool) error {
		if required[stageName] {
			return nil
		}
		st, ok := stageDefs[stageName]
		if !ok {
			return fmt.Errorf("stage not found: %s", stageName)
		}
		if visiting[stageName] {
			return fmt.Errorf("cyclic stage dependency detected at %s", stageName)
		}

		visiting[stageName] = true
		for _, depName := range st.DependsOn {
			if err := collectDeps(depName, visiting); err != nil {
				return err
			}
		}
		delete(visiting, stageName)
		required[stageName] = true
		return nil
	}

	if wantAll {
		for _, st := range m.BuildPipeline.Stages {
			required[st.Name] = true
		}
	} else {
		for _, s := range requested {
			stageName := normalizeStageRequest(s)
			if stageName == "" {
				continue
			}
			if err := collectDeps(stageName, map[string]bool{}); err != nil {
				return nil, err
			}
		}
	}

	var stages []matrix.Stage
	for _, st := range m.BuildPipeline.Stages {
		if required[st.Name] {
			stages = append(stages, st)
		}
	}

	if len(stages) == 0 {
		return nil, fmt.Errorf("no stages selected")
	}

	return stages, nil
}

func normalizeStageRequest(stageName string) string {
	stageName = strings.TrimSpace(stageName)
	switch strings.ToLower(stageName) {
	case "base":
		return "base_test"
	case "app":
		return "app_test"
	case "test":
		return "app_test"
	default:
		return stageName
	}
}

func resolveDockerfileOverride(requested []string, dockerfile string) (string, string, error) {
	dockerfile = strings.TrimSpace(dockerfile)
	if dockerfile == "" {
		return "", "", nil
	}

	var explicit []string
	for _, s := range requested {
		stageName := strings.TrimSpace(s)
		if stageName == "" {
			continue
		}
		if strings.EqualFold(stageName, "all") {
			return "", "", fmt.Errorf("--dockerfile requires exactly one explicit --stage")
		}
		explicit = append(explicit, stageName)
	}
	if len(explicit) != 1 {
		return "", "", fmt.Errorf("--dockerfile requires exactly one explicit --stage")
	}
	return explicit[0], dockerfile, nil
}

func buildStageTask(
	stageName string,
	u BuildUnit,
	kanikoImage string,
	contextDir string,
	regPrefix string,
	cacheEnabled bool,
	cacheRepoTpl string,
	buildCache matrix.Cache,
	overrideStageName string,
	overrideDockerfile string,
) Task {
	taskName := taskNameForStage(stageName, u)
	cacheRepo := applyRepoTemplate(cacheRepoTpl, regPrefix, u.Hardware, stageName)
	task := Task{
		Name:     taskName,
		Stage:    stageName,
		Hardware: u.Hardware,
		App:      fmt.Sprintf("%s%s-%s", u.AppName, u.AppVersion, u.VariantName),
		Kaniko: KanikoSpec{
			Image:      kanikoImage,
			ContextDir: contextDir,
			Cache: CacheSpec{
				Enabled:     cacheEnabled,
				Repo:        cacheRepo,
				TTL:         buildCache.TTL,
				KeyTemplate: buildCache.KeyTemplate,
			},
		},
	}

	switch stageName {
	case "host_driver":
		task.HostDriver = buildHostDriverSpec(u)
	case "base_image":
		task.Kaniko.Dockerfile = fmt.Sprintf("dockerfiles/base_image/%s/%s/%s/Dockerfile", u.Hardware, u.BaseRef, u.BaseVariant.TagSuffix)
		task.Kaniko.Destination = u.BaseImageDest
		task.Kaniko.BuildArgs = map[string]string{}
		for k, v := range u.BaseBuildArgs {
			task.Kaniko.BuildArgs[k] = v
		}
		task.Kaniko.BuildArgs["BASE_IMAGE"] = u.BaseSourceImage
	case "base_test":
		task.TestImage = u.BaseImageDest
		task.TestCommands = []string{
			baseRuntimeSmokeCommand(u),
		}
		applyTestResources(&task, u)
	case "app_image":
		task.Kaniko.Dockerfile = fmt.Sprintf("dockerfiles/app_image/%s/%s/%s/%s/Dockerfile", u.Hardware, u.AppName, u.AppVersion, sanitizeName(u.VariantName))
		task.Kaniko.Destination = u.AppImageDest
		task.Kaniko.BuildArgs = map[string]string{}
		for k, v := range u.BuildArgs {
			task.Kaniko.BuildArgs[k] = v
		}
		task.Kaniko.BuildArgs["BASE_IMAGE"] = u.BaseImageDest
	case "app_test":
		task.TestImage = u.AppImageDest
		task.TestCommands = []string{
			fmt.Sprintf("if [ -f /opt/k8ace/hack/test/smoke.sh ]; then bash /opt/k8ace/hack/test/smoke.sh L2 %s %s; else python3 --version && (pip list --format=columns 2>/dev/null || pip3 list --format=columns 2>/dev/null || true); fi", u.AppName, u.Hardware),
		}
		applyTestResources(&task, u)
		task.Kaniko.Image = u.AppImageDest
		task.Kaniko.NoPush = true
		task.Kaniko.Cache.Enabled = false
		task.Kaniko.Cache.Repo = ""
	default:
		task.Kaniko.Image = "alpine:3.20"
		task.Kaniko.NoPush = true
		task.Kaniko.Cache.Enabled = false
		task.Kaniko.Cache.Repo = ""
	}

	if overrideDockerfile != "" && stageName == overrideStageName {
		task.Kaniko.Dockerfile = overrideDockerfile
	}

	return task
}

func taskNameForStage(stageName string, u BuildUnit) string {
	switch stageName {
	case "base_image":
		return sanitizeName(strings.Join([]string{"base-image", u.Hardware, u.BaseRef, u.BaseVariant.TagSuffix}, "-"))
	case "base_test":
		return sanitizeName(strings.Join([]string{"base-test", u.Hardware, u.BaseRef, u.BaseVariant.TagSuffix}, "-"))
	case "app_image":
		return sanitizeName(strings.Join([]string{"app-image", u.Hardware, u.AppName, u.AppVersion, u.VariantName}, "-"))
	case "app_test":
		return sanitizeName(strings.Join([]string{"app-test", u.Hardware, u.AppName, u.AppVersion, u.VariantName}, "-"))
	default:
		return sanitizeName(strings.Join([]string{stageName, u.Hardware, u.AppName, u.AppVersion, u.VariantName}, "-"))
	}
}

func applyTestResources(task *Task, u BuildUnit) {
	if strings.EqualFold(strings.TrimSpace(u.Hardware), "nvidia") {
		task.TestResourceLimits = map[string]string{"nvidia.com/gpu": "1"}
		task.TestResourceRequests = map[string]string{"nvidia.com/gpu": "1"}
	}
}

func baseRuntimeSmokeCommand(u BuildUnit) string {
	if !strings.EqualFold(strings.TrimSpace(u.Hardware), "nvidia") {
		return fmt.Sprintf("echo '[base_test] unsupported hardware=%s'; exit 1", shellQuote(u.Hardware))
	}

	return `set -eu
echo "[base_test] NVIDIA CUDA runtime smoke"
python3 --version
(pip --version || pip3 --version)
command -v nvidia-smi
nvidia-smi -L | grep -q '^GPU '
nvidia-smi --query-gpu=driver_version --format=csv,noheader | head -n1 | grep -Eq '^[0-9]+(\.[0-9]+)+'
ldconfig -p 2>/dev/null | grep -q 'libcuda.so.1' || test -e /usr/lib/x86_64-linux-gnu/libcuda.so.1 || test -e /usr/local/cuda/compat/libcuda.so.1
ldconfig -p 2>/dev/null | grep -Eq 'libcudart\.so|libcuda\.so' || find /usr /opt /workspace \( -name 'libcudart.so*' -o -name 'libcuda.so*' \) 2>/dev/null | head -n1 | grep -q .
echo "[base_test] PASS: base image can see NVIDIA CUDA runtime"
`
}

func stageByName(stages []matrix.Stage, stageName string) (matrix.Stage, bool) {
	for _, st := range stages {
		if st.Name == stageName {
			return st, true
		}
	}
	return matrix.Stage{}, false
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
