package render

type ObjectMeta struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

type WorkflowTemplate struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   ObjectMeta   `yaml:"metadata"`
	Spec       WorkflowSpec `yaml:"spec"`
}

type Workflow struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   ObjectMeta   `yaml:"metadata"`
	Spec       WorkflowSpec `yaml:"spec"`
}

type WorkflowSpec struct {
	Entrypoint         string     `yaml:"entrypoint"`
	ServiceAccountName string     `yaml:"serviceAccountName,omitempty"`
	Parallelism        int        `yaml:"parallelism,omitempty"`
	Templates          []Template `yaml:"templates"`
	Arguments          *Arguments `yaml:"arguments,omitempty"`
	Volumes            []Volume   `yaml:"volumes,omitempty"`
}

type Template struct {
	Name      string       `yaml:"name"`
	DAG       *DAGTemplate `yaml:"dag,omitempty"`
	Container *Container   `yaml:"container,omitempty"`
	Inputs    *Inputs      `yaml:"inputs,omitempty"`
}

type DAGTemplate struct {
	Tasks []DAGTask `yaml:"tasks"`
}

type DAGTask struct {
	Name         string     `yaml:"name"`
	Template     string     `yaml:"template"`
	Dependencies []string   `yaml:"dependencies,omitempty"`
	Arguments    *Arguments `yaml:"arguments,omitempty"`
}

type Inputs struct {
	Parameters []Parameter `yaml:"parameters,omitempty"`
}

type Arguments struct {
	Parameters []Parameter `yaml:"parameters,omitempty"`
}

type Parameter struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value,omitempty"`
}

type Container struct {
	Image        string        `yaml:"image"`
	Command      []string      `yaml:"command,omitempty"`
	Args         []string      `yaml:"args,omitempty"`
	Env          []EnvVar      `yaml:"env,omitempty"`
	VolumeMounts []VolumeMount `yaml:"volumeMounts,omitempty"`
}

type EnvVar struct {
	Name      string        `yaml:"name"`
	Value     string        `yaml:"value,omitempty"`
	ValueFrom *EnvVarSource `yaml:"valueFrom,omitempty"`
}

type EnvVarSource struct {
	SecretKeyRef *SecretKeySelector `yaml:"secretKeyRef,omitempty"`
}

type SecretKeySelector struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

type VolumeMount struct {
	Name      string `yaml:"name"`
	MountPath string `yaml:"mountPath"`
	ReadOnly  bool   `yaml:"readOnly,omitempty"`
}

type Volume struct {
	Name   string              `yaml:"name"`
	Secret *SecretVolumeSource `yaml:"secret,omitempty"`
}

type SecretVolumeSource struct {
	SecretName string      `yaml:"secretName"`
	Items      []KeyToPath `yaml:"items,omitempty"`
}

type KeyToPath struct {
	Key  string `yaml:"key"`
	Path string `yaml:"path"`
}
