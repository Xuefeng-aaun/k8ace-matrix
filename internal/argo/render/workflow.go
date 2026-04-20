package render

import (
	"fmt"
	"sort"
	"strings"

	"k8ace-matrix/internal/pipeline"
)

type Options struct {
	Kind               string
	Name               string
	Namespace          string
	ServiceAccountName string
	RegistrySecretName string
	Labels             map[string]string
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
	templates = append(templates, kanikoTemplates(p.Tasks, opt.RegistrySecretName)...)

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

func kanikoTemplates(tasks []pipeline.Task, registrySecretName string) []Template {
	mounts := kanikoVolumeMounts(registrySecretName)

	out := make([]Template, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, Template{
			Name: task.Name,
			Container: &Container{
				Image: task.Kaniko.Image,
				Command: []string{
					"/kaniko/executor",
				},
				Args:         buildKanikoCommandArgs(task.Kaniko),
				VolumeMounts: mounts,
			},
		})
	}
	return out
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

func buildKanikoCommandArgs(k pipeline.KanikoSpec) []string {
	args := []string{
		"--context=" + k.ContextDir,
		"--dockerfile=" + k.Dockerfile,
		"--destination=" + k.Destination,
	}
	args = append(args, buildKanikoArgs(k)...)
	return args
}

func buildKanikoArgs(k pipeline.KanikoSpec) []string {
	var args []string

	if k.Cache.Enabled {
		args = append(args, "--cache=true")
		if k.Cache.Repo != "" {
			args = append(args, "--cache-repo="+k.Cache.Repo)
		}
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
