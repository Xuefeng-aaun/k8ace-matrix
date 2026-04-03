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

	spec := WorkflowSpec{
		Entrypoint:         "main",
		ServiceAccountName: opt.ServiceAccountName,
		Volumes:            volumes(opt),
		Templates: []Template{
			{
				Name: "main",
				DAG: &DAGTemplate{
					Tasks: dagTasks(p),
				},
			},
			kanikoTemplate(opt.RegistrySecretName),
		},
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
		args := buildKanikoArgs(t.Kaniko)
		out = append(out, DAGTask{
			Name:         t.Name,
			Template:     "kaniko",
			Dependencies: t.DependsOn,
			Arguments: &Arguments{
				Parameters: []Parameter{
					{Name: "context", Value: t.Kaniko.ContextDir},
					{Name: "dockerfile", Value: t.Kaniko.Dockerfile},
					{Name: "destination", Value: t.Kaniko.Destination},
					{Name: "extraArgs", Value: strings.Join(args, " ")},
					{Name: "kanikoImage", Value: t.Kaniko.Image},
				},
			},
		})
	}
	return out
}

func kanikoTemplate(registrySecretName string) Template {
	var mounts []VolumeMount
	if strings.TrimSpace(registrySecretName) != "" {
		mounts = append(mounts, VolumeMount{
			Name:      "registry-config",
			MountPath: "/kaniko/.docker",
			ReadOnly:  true,
		})
	}

	return Template{
		Name: "kaniko",
		Inputs: &Inputs{
			Parameters: []Parameter{
				{Name: "context"},
				{Name: "dockerfile"},
				{Name: "destination"},
				{Name: "extraArgs"},
				{Name: "kanikoImage"},
			},
		},
		Container: &Container{
			Image: "{{inputs.parameters.kanikoImage}}",
			Command: []string{
				"/kaniko/executor",
			},
			Args: []string{
				"--context={{inputs.parameters.context}}",
				"--dockerfile={{inputs.parameters.dockerfile}}",
				"--destination={{inputs.parameters.destination}}",
				"{{inputs.parameters.extraArgs}}",
			},
			VolumeMounts: mounts,
		},
	}
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
