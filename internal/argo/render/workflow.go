package render

import (
	"fmt"
	"sort"
	"strings"

	"k8ace-matrix/internal/pipeline"
)

type Options struct {
	Kind                    string
	Name                    string
	Namespace               string
	ServiceAccountName      string
	Parallelism             int
	ContextEnv              []EnvVar
	RegistryMirrors         []string
	InsecureRegistries      []string
	SkipPushPermissionCheck bool
	RegistrySecretName      string
	Labels                  map[string]string
}

func BuildWorkflowYAML(p *pipeline.Plan, opt Options) ([]byte, error) {
	kind := strings.ToLower(strings.TrimSpace(opt.Kind))
	if kind == "" {
		kind = "workflowtemplate"
	}
	if kind != "workflowtemplate" && kind != "workflow" {
		return nil, fmt.Errorf("unsupported kind: %s", opt.Kind)
	}

	name := strings.TrimSpace(opt.Name)
	if name == "" {
		name = p.Name
	}

	templates := []Template{
		{
			Name: "main",
			DAG: &DAGTemplate{
				Tasks: dagTasks(p),
			},
		},
	}
	templates = append(templates, taskTemplates(p.Tasks, opt.ContextEnv, opt.RegistrySecretName, opt.RegistryMirrors, opt.InsecureRegistries, opt.SkipPushPermissionCheck)...)

	spec := WorkflowSpec{
		Entrypoint:         "main",
		ServiceAccountName: opt.ServiceAccountName,
		Parallelism:        opt.Parallelism,
		Volumes:            volumes(opt),
		Templates:          templates,
	}

	var out any
	if kind == "workflowtemplate" {
		out = WorkflowTemplate{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "WorkflowTemplate",
			Metadata: ObjectMeta{
				Name:      name,
				Namespace: opt.Namespace,
				Labels:    opt.Labels,
			},
			Spec: spec,
		}
	} else {
		out = Workflow{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
			Metadata: ObjectMeta{
				Name:      name,
				Namespace: opt.Namespace,
				Labels:    opt.Labels,
			},
			Spec: spec,
		}
	}

	return MarshalYAML(out)
}

func dagTasks(p *pipeline.Plan) []DAGTask {
	var out []DAGTask
	for _, t := range p.Tasks {
		out = append(out, DAGTask{
			Name:         t.Name,
			Template:     t.Name,
			Dependencies: t.DependsOn,
		})
	}
	return out
}

func taskTemplates(tasks []pipeline.Task, contextEnv []EnvVar, registrySecretName string, registryMirrors []string, insecureRegistries []string, skipPushPermissionCheck bool) []Template {
	mounts := kanikoVolumeMounts(registrySecretName)

	out := make([]Template, 0, len(tasks))
	for _, task := range tasks {
		if len(task.HostDriver.Commands) > 0 {
			out = append(out, hostDriverTemplate(task))
			continue
		}

		if task.TestImage != "" {
			out = append(out, testTemplate(task))
			continue
		}

		if strings.TrimSpace(task.Kaniko.Dockerfile) == "" {
			out = append(out, noopTemplate(task))
			continue
		}

		out = append(out, Template{
			Name: task.Name,
			Container: &Container{
				Image: task.Kaniko.Image,
				Command: []string{
					"/kaniko/executor",
				},
				Args:         buildKanikoCommandArgs(task.Kaniko, registryMirrors, insecureRegistries, skipPushPermissionCheck, contextEnv),
				Env:          cloneEnvVars(contextEnv),
				VolumeMounts: mounts,
			},
		})
	}
	return out
}

func hostDriverTemplate(task pipeline.Task) Template {
	script := fmt.Sprintf("echo '[host_driver] stage=%s task=%s';\n", task.Stage, task.Name)
	for _, cmd := range task.HostDriver.Commands {
		script += cmd + "\n"
	}

	return Template{
		Name: task.Name,
		Container: &Container{
			Image:   firstNonEmptyImage(task.HostDriver.Image, "alpine:3.20"),
			Command: []string{"sh", "-c"},
			Args:    []string{script},
			Resources: resourceRequirements(
				task.HostDriver.ResourceLimits,
				task.HostDriver.ResourceRequests,
			),
		},
	}
}

func noopTemplate(task pipeline.Task) Template {
	return Template{
		Name: task.Name,
		Container: &Container{
			Image: firstNonEmptyImage(task.Kaniko.Image, "alpine:3.20"),
			Command: []string{
				"sh",
				"-c",
			},
			Args: []string{
				fmt.Sprintf("echo '[noop] stage=%s task=%s';", task.Stage, task.Name),
			},
		},
	}
}

func resourceRequirements(limits, requests map[string]string) *Resources {
	if len(limits) == 0 && len(requests) == 0 {
		return nil
	}
	return &Resources{
		Limits:   cleanResourceMap(limits),
		Requests: cleanResourceMap(requests),
	}
}

func cleanResourceMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func testTemplate(task pipeline.Task) Template {
	label := strings.TrimSpace(task.Stage)
	if label == "" {
		label = "test"
	}

	script := fmt.Sprintf("echo '[%s] task=%s image=%s';\n", label, task.Name, task.TestImage)
	for idx, cmd := range task.TestCommands {
		checkNo := idx + 1
		script += fmt.Sprintf("echo '[%s] running check %d';\n", label, checkNo)
		script += "(\n" + cmd + "\n) || { echo '[" + label + "] FAILED check " + fmt.Sprint(checkNo) + "'; exit 1; };\n"
	}
	script += fmt.Sprintf("echo '[%s] ALL PASSED';\n", label)

	return Template{
		Name: task.Name,
		Container: &Container{
			Image:   task.TestImage,
			Command: []string{"sh", "-c"},
			Args:    []string{script},
			Resources: resourceRequirements(
				task.TestResourceLimits,
				task.TestResourceRequests,
			),
		},
	}
}

func firstNonEmptyImage(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func kanikoVolumeMounts(registrySecretName string) []VolumeMount {
	if strings.TrimSpace(registrySecretName) == "" {
		return nil
	}

	return []VolumeMount{
		{
			Name:      "registry-config",
			MountPath: "/kaniko/.docker",
			ReadOnly:  true,
		},
	}
}

func buildKanikoCommandArgs(k pipeline.KanikoSpec, registryMirrors []string, insecureRegistries []string, skipPushPermissionCheck bool, contextEnv []EnvVar) []string {
	args := []string{
		"--context=" + k.ContextDir,
		"--dockerfile=" + k.Dockerfile,
		"--destination=" + k.Destination,
	}
	args = append(args, buildKanikoArgs(k, registryMirrors, insecureRegistries, skipPushPermissionCheck)...)
	args = append(args, buildProxyBuildArgs(k.BuildArgs, contextEnv)...)
	return args
}

func buildKanikoArgs(k pipeline.KanikoSpec, registryMirrors []string, insecureRegistries []string, skipPushPermissionCheck bool) []string {
	var args []string

	if k.Cache.Enabled {
		args = append(args, "--cache=true")
	}
	if skipPushPermissionCheck {
		args = append(args, "--skip-push-permission-check")
	}
	args = append(args, buildRegistryMirrorArgs(registryMirrors)...)
	args = append(args, buildInsecureRegistryArgs(insecureRegistries)...)
	if k.Cache.Enabled && k.Cache.Repo != "" {
		args = append(args, "--cache-repo="+k.Cache.Repo)
	}
	if k.NoPush {
		args = append(args, "--no-push")
	}

	keys := make([]string, 0, len(k.BuildArgs))
	for k := range k.BuildArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, "--build-arg="+key+"="+k.BuildArgs[key])
	}

	return args
}

func buildRegistryMirrorArgs(registryMirrors []string) []string {
	return buildMultiValueArgs("--registry-mirror=", registryMirrors)
}

func buildInsecureRegistryArgs(insecureRegistries []string) []string {
	return buildMultiValueArgs("--insecure-registry=", insecureRegistries)
}

func buildMultiValueArgs(prefix string, values []string) []string {
	var args []string
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		args = append(args, prefix+value)
	}
	return args
}

func buildProxyBuildArgs(existing map[string]string, contextEnv []EnvVar) []string {
	proxyKeys := map[string]bool{
		"HTTP_PROXY":  true,
		"HTTPS_PROXY": true,
		"http_proxy":  true,
		"https_proxy": true,
		"NO_PROXY":    true,
		"no_proxy":    true,
	}

	var args []string
	for _, env := range contextEnv {
		key := strings.TrimSpace(env.Name)
		value := strings.TrimSpace(env.Value)
		if !proxyKeys[key] || value == "" {
			continue
		}
		if _, ok := existing[key]; ok {
			continue
		}
		args = append(args, "--build-arg="+key+"="+value)
	}
	return args
}

func BuildContextEnvVars(staticEnv map[string]string, secretName string, secretEnv map[string]string) ([]EnvVar, error) {
	var envs []EnvVar

	keys := sortedNonEmptyKeys(staticEnv)
	for _, key := range keys {
		envs = append(envs, EnvVar{
			Name:  key,
			Value: strings.TrimSpace(staticEnv[key]),
		})
	}

	secretKeys := sortedNonEmptyKeys(secretEnv)
	if len(secretKeys) > 0 && strings.TrimSpace(secretName) == "" {
		return nil, fmt.Errorf("build context secret_env requires secret_name")
	}
	for _, key := range secretKeys {
		envs = append(envs, EnvVar{
			Name: key,
			ValueFrom: &EnvVarSource{
				SecretKeyRef: &SecretKeySelector{
					Name: strings.TrimSpace(secretName),
					Key:  strings.TrimSpace(secretEnv[key]),
				},
			},
		})
	}

	return envs, nil
}

func sortedNonEmptyKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key, value := range m {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneEnvVars(envs []EnvVar) []EnvVar {
	if len(envs) == 0 {
		return nil
	}

	out := make([]EnvVar, 0, len(envs))
	for _, env := range envs {
		copied := EnvVar{
			Name:  env.Name,
			Value: env.Value,
		}
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
			copied.ValueFrom = &EnvVarSource{
				SecretKeyRef: &SecretKeySelector{
					Name: env.ValueFrom.SecretKeyRef.Name,
					Key:  env.ValueFrom.SecretKeyRef.Key,
				},
			}
		}
		out = append(out, copied)
	}

	return out
}

func volumes(opt Options) []Volume {
	if strings.TrimSpace(opt.RegistrySecretName) == "" {
		return nil
	}
	return []Volume{
		{
			Name: "registry-config",
			Secret: &SecretVolumeSource{
				SecretName: opt.RegistrySecretName,
				Items: []KeyToPath{
					{Key: ".dockerconfigjson", Path: "config.json"},
				},
			},
		},
	}
}
