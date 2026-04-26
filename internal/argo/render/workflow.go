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
	ContextEnv              []EnvVar
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
	templates = append(templates, taskTemplates(p.Tasks, opt.ContextEnv, opt.RegistrySecretName, opt.InsecureRegistries, opt.SkipPushPermissionCheck)...)

	spec := WorkflowSpec{
		Entrypoint:         "main",
		ServiceAccountName: opt.ServiceAccountName,
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

func taskTemplates(tasks []pipeline.Task, contextEnv []EnvVar, registrySecretName string, insecureRegistries []string, skipPushPermissionCheck bool) []Template {
	mounts := kanikoVolumeMounts(registrySecretName)

	out := make([]Template, 0, len(tasks))
	for _, task := range tasks {
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
				Args:         buildKanikoCommandArgs(task.Kaniko, insecureRegistries, skipPushPermissionCheck),
				Env:          cloneEnvVars(contextEnv),
				VolumeMounts: mounts,
			},
		})
	}
	return out
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

func buildKanikoCommandArgs(k pipeline.KanikoSpec, insecureRegistries []string, skipPushPermissionCheck bool) []string {
	args := []string{
		"--context=" + k.ContextDir,
		"--dockerfile=" + k.Dockerfile,
		"--destination=" + k.Destination,
	}
	args = append(args, buildKanikoArgs(k, insecureRegistries, skipPushPermissionCheck)...)
	return args
}

func buildKanikoArgs(k pipeline.KanikoSpec, insecureRegistries []string, skipPushPermissionCheck bool) []string {
	var args []string

	if k.Cache.Enabled {
		args = append(args, "--cache=true")
	}
	if skipPushPermissionCheck {
		args = append(args, "--skip-push-permission-check")
	}
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
