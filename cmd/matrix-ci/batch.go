package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"k8ace-matrix/internal/argo/render"
	"k8ace-matrix/internal/matrix"
	"k8ace-matrix/internal/pipeline"
)

type batchRow struct {
	Hardware   string
	AppName    string
	AppVersion string
	Variant    string
	Stages     []string
}

func newBatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Render/apply/create workflows from a batch plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			planPath, _ := cmd.Flags().GetString("plan")
			if strings.TrimSpace(planPath) == "" {
				return fmt.Errorf("--plan is required")
			}

			matrixPath, _ := cmd.Flags().GetString("matrix")
			if strings.TrimSpace(matrixPath) == "" {
				if _, err := os.Stat("images-matrix.yaml"); err == nil {
					matrixPath = "images-matrix.yaml"
				} else {
					return fmt.Errorf("--matrix is required")
				}
			}

			m, err := matrix.Load(matrixPath)
			if err != nil {
				return err
			}

			rows, err := readBatchPlan(planPath)
			if err != nil {
				return err
			}

			outDir, _ := cmd.Flags().GetString("out-dir")
			if strings.TrimSpace(outDir) == "" {
				outDir = "dist/argo"
			}

			apply, _ := cmd.Flags().GetBool("apply")
			create, _ := cmd.Flags().GetBool("create")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			wait, _ := cmd.Flags().GetBool("wait")
			waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
			var createdWorkflows []workflowRef

			aw := m.CICD.ArgoWorkflows
			contextFlagChanged := cmd.Flags().Changed("context")
			contextDir, _ := cmd.Flags().GetString("context")
			contextDir = resolveContextDir(contextDir, contextFlagChanged, aw)

			namespace, _ := cmd.Flags().GetString("namespace")
			namespace = firstNonEmpty(namespace, aw.Namespace)
			serviceAccount, _ := cmd.Flags().GetString("service-account")
			registrySecretName, _ := cmd.Flags().GetString("registry-secret-name")
			registrySecretName = firstNonEmpty(registrySecretName, aw.RegistrySecret)
			labels, _ := cmd.Flags().GetStringSlice("label")

			contextEnv, err := buildContextEnv(aw)
			if err != nil {
				return err
			}

			for _, row := range rows {
				paths, err := renderBatchRow(cmd, m, row, outDir, contextDir, contextEnv, namespace, serviceAccount, registrySecretName, labels)
				if err != nil {
					return err
				}

				for _, p := range paths {
					if apply || create {
						if err := runKubectl(dryRun, "apply", "-f", p.templatePath); err != nil {
							return err
						}
					}
					if create {
						workflowName, workflowNamespace, err := createWorkflow(dryRun, p.workflowPath, namespace)
						if err != nil {
							return err
						}
						if !dryRun {
							createdWorkflows = append(createdWorkflows, workflowRef{
								Name:      workflowName,
								Namespace: workflowNamespace,
							})
						}
					}
				}
			}

			if create && wait && !dryRun {
				return waitForWorkflows(createdWorkflows, waitTimeout)
			}

			return nil
		},
	}

	cmd.Flags().String("plan", "", "batch plan path")
	cmd.Flags().String("matrix", "", "path to images-matrix.yaml")
	cmd.Flags().String("out-dir", "dist/argo", "output directory")
	cmd.Flags().Bool("apply", false, "apply generated WorkflowTemplates with kubectl")
	cmd.Flags().Bool("create", false, "apply WorkflowTemplates and create Workflows")
	cmd.Flags().Bool("dry-run", false, "print kubectl commands instead of executing them")
	cmd.Flags().Bool("wait", true, "wait for created Workflows and print stage feedback")
	cmd.Flags().Duration("wait-timeout", 0, "maximum time to wait for each Workflow, 0 means no timeout")

	cmd.Flags().String("namespace", "", "kubernetes namespace")
	cmd.Flags().String("service-account", "", "argo service account name")
	cmd.Flags().StringSlice("label", nil, "extra labels k=v (repeatable)")
	cmd.Flags().String("registry-secret-name", "", "k8s secret name for docker registry credentials (dockerconfigjson)")

	cmd.Flags().String("builder", "kaniko", "builder engine")
	cmd.Flags().String("context", "", "build context override (defaults to ci_cd.argo_workflows.build_context.default or '.')")
	cmd.Flags().String("dockerfile", "", "dockerfile path override")
	cmd.Flags().String("kaniko-image", "", "kaniko executor image")
	cmd.Flags().String("registry-prefix", "", "registry prefix override")
	cmd.Flags().String("version-suffix", "dev", "image tag/version suffix")

	return cmd
}

func readBatchPlan(path string) ([]batchRow, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rows []batchRow
	for idx, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		cols := strings.Fields(line)
		if len(cols) < 4 {
			return nil, fmt.Errorf("%s:%d requires at least 4 columns: hardware app_name app_version variant [stages]", path, idx+1)
		}

		stages := []string{"all"}
		if len(cols) >= 5 && strings.TrimSpace(cols[4]) != "" {
			stages = splitCommaList(cols[4])
		}

		rows = append(rows, batchRow{
			Hardware:   cols[0],
			AppName:    cols[1],
			AppVersion: cols[2],
			Variant:    cols[3],
			Stages:     stages,
		})
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("batch plan is empty: %s", path)
	}
	return rows, nil
}

type generatedBatchPaths struct {
	templatePath string
	workflowPath string
}

func renderBatchRow(
	cmd *cobra.Command,
	m *matrix.Matrix,
	row batchRow,
	outDir string,
	contextDir string,
	contextEnv []render.EnvVar,
	namespace string,
	serviceAccount string,
	registrySecretName string,
	labels []string,
) ([]generatedBatchPaths, error) {
	aw := m.CICD.ArgoWorkflows

	registryPrefix, _ := cmd.Flags().GetString("registry-prefix")
	versionSuffix, _ := cmd.Flags().GetString("version-suffix")
	builder, _ := cmd.Flags().GetString("builder")
	dockerfile, _ := cmd.Flags().GetString("dockerfile")
	kanikoImage, _ := cmd.Flags().GetString("kaniko-image")
	if strings.TrimSpace(kanikoImage) == "" {
		kanikoImage = firstNonEmpty(aw.KanikoImage, "gcr.io/kaniko-project/executor:latest")
	}

	sel := pipeline.Selection{
		Hardwares:      []string{row.Hardware},
		AppName:        row.AppName,
		AppVersion:     row.AppVersion,
		Variant:        row.Variant,
		Stages:         row.Stages,
		RegistryPrefix: registryPrefix,
		VersionSuffix:  versionSuffix,
		Builder:        builder,
		ContextDir:     contextDir,
		Dockerfile:     dockerfile,
		KanikoImage:    kanikoImage,
	}

	plans, err := pipeline.BuildPlans(m, sel)
	if err != nil {
		return nil, err
	}

	var out []generatedBatchPaths
	for _, p := range plans {
		name := p.Name
		if suffix := stagesNameSuffix(row.Stages); suffix != "" {
			name = name + "-" + suffix
		}

		yamlBytes, err := render.BuildWorkflowYAML(p, render.Options{
			Kind:                    "workflowtemplate",
			Name:                    name,
			Namespace:               namespace,
			ServiceAccountName:      firstNonEmpty(serviceAccount, aw.ServiceAccount),
			Parallelism:             aw.Parallelism,
			ContextEnv:              contextEnv,
			RegistryMirrors:         aw.RegistryMirrors,
			InsecureRegistries:      aw.InsecureRegistries,
			Kaniko:                  aw.Kaniko,
			SkipPushPermissionCheck: aw.SkipPushPermissionCheck,
			RegistrySecretName:      registrySecretName,
			Labels:                  labelsFromCSV(labels),
		})
		if err != nil {
			return nil, err
		}

		templatePath := filepath.Join(outDir, "workflowtemplates", name+".yaml")
		if err := writeFile(templatePath, yamlBytes); err != nil {
			return nil, err
		}

		workflowPath := filepath.Join(outDir, "workflows", name+"-workflow.yaml")
		if err := writeFile(workflowPath, workflowRefYAML(namespace, name)); err != nil {
			return nil, err
		}

		out = append(out, generatedBatchPaths{
			templatePath: templatePath,
			workflowPath: workflowPath,
		})
	}

	return out, nil
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func workflowRefYAML(namespace, templateName string) []byte {
	nsLine := ""
	if strings.TrimSpace(namespace) != "" {
		nsLine = "  namespace: " + strings.TrimSpace(namespace) + "\n"
	}

	return []byte(fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: %s-
%sspec:
  workflowTemplateRef:
    name: %s
`, templateName, nsLine, templateName))
}

func splitCommaList(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return []string{"all"}
	}
	return out
}

func stagesNameSuffix(stages []string) string {
	if len(stages) == 0 {
		return ""
	}
	var parts []string
	for _, stage := range stages {
		stage = strings.TrimSpace(stage)
		if stage == "" || strings.EqualFold(stage, "all") {
			continue
		}
		parts = append(parts, strings.ReplaceAll(stage, "_", "-"))
	}
	return strings.Join(parts, "-")
}

func runKubectl(dryRun bool, args ...string) error {
	if dryRun {
		fmt.Println("kubectl " + strings.Join(args, " "))
		return nil
	}

	c := exec.Command("kubectl", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

type createdWorkflow struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
}

type workflowRef struct {
	Name      string
	Namespace string
}

type workflowDocument struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Status workflowStatus `json:"status"`
}

func waitForWorkflows(workflows []workflowRef, timeout time.Duration) error {
	var firstErr error
	for _, wf := range workflows {
		if err := waitForWorkflow(wf.Namespace, wf.Name, timeout); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

type workflowStatus struct {
	Phase   string                  `json:"phase"`
	Message string                  `json:"message"`
	Nodes   map[string]workflowNode `json:"nodes"`
}

type workflowNode struct {
	DisplayName  string `json:"displayName"`
	TemplateName string `json:"templateName"`
	Type         string `json:"type"`
	Phase        string `json:"phase"`
	Message      string `json:"message"`
	StartedAt    string `json:"startedAt"`
	FinishedAt   string `json:"finishedAt"`
}

func createWorkflow(dryRun bool, workflowPath string, namespace string) (string, string, error) {
	if dryRun {
		fmt.Println("kubectl create -f " + workflowPath + " -o json")
		return "", "", nil
	}

	c := exec.Command("kubectl", "create", "-f", workflowPath, "-o", "json")
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	out, err := c.Output()
	if err != nil {
		return "", "", err
	}

	var wf createdWorkflow
	if err := json.Unmarshal(out, &wf); err != nil {
		return "", "", fmt.Errorf("parse kubectl create output: %w", err)
	}
	if strings.TrimSpace(wf.Metadata.Name) == "" {
		return "", "", fmt.Errorf("kubectl create did not return workflow metadata.name")
	}

	wfNamespace := firstNonEmpty(wf.Metadata.Namespace, namespace)
	fmt.Printf("workflow.argoproj.io/%s created\n", wf.Metadata.Name)
	return wf.Metadata.Name, wfNamespace, nil
}

func waitForWorkflow(namespace string, workflowName string, timeout time.Duration) error {
	namespace = strings.TrimSpace(namespace)
	workflowName = strings.TrimSpace(workflowName)
	if workflowName == "" {
		return fmt.Errorf("workflow name is empty")
	}

	fmt.Printf("waiting for workflow %s ...\n", workflowName)
	started := time.Now()
	lastPhase := ""
	var lastDoc *workflowDocument

	for {
		doc, err := getWorkflow(namespace, workflowName)
		if err != nil {
			return err
		}
		lastDoc = doc

		phase := strings.TrimSpace(doc.Status.Phase)
		if phase == "" {
			phase = "Pending"
		}
		if phase != lastPhase {
			fmt.Printf("workflow %s phase=%s\n", workflowName, phase)
			lastPhase = phase
		}

		switch phase {
		case "Succeeded":
			printWorkflowSummary(doc)
			fmt.Printf("PASS: workflow %s completed successfully\n", workflowName)
			return nil
		case "Failed", "Error":
			printWorkflowSummary(doc)
			return fmt.Errorf("workflow %s ended with phase=%s", workflowName, phase)
		}

		if timeout > 0 && time.Since(started) > timeout {
			if lastDoc != nil {
				printWorkflowSummary(lastDoc)
			}
			return fmt.Errorf("workflow %s wait timeout after %s", workflowName, timeout)
		}

		time.Sleep(15 * time.Second)
	}
}

func getWorkflow(namespace string, workflowName string) (*workflowDocument, error) {
	args := []string{}
	if strings.TrimSpace(namespace) != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, "get", "workflow", workflowName, "-o", "json")

	c := exec.Command("kubectl", args...)
	c.Stderr = os.Stderr
	out, err := c.Output()
	if err != nil {
		return nil, err
	}

	var doc workflowDocument
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("parse workflow status: %w", err)
	}
	return &doc, nil
}

func printWorkflowSummary(doc *workflowDocument) {
	name := firstNonEmpty(doc.Metadata.Name, "<unknown>")
	namespace := strings.TrimSpace(doc.Metadata.Namespace)
	if namespace != "" {
		fmt.Printf("workflow summary: %s/%s phase=%s\n", namespace, name, firstNonEmpty(doc.Status.Phase, "Unknown"))
	} else {
		fmt.Printf("workflow summary: %s phase=%s\n", name, firstNonEmpty(doc.Status.Phase, "Unknown"))
	}
	if strings.TrimSpace(doc.Status.Message) != "" {
		fmt.Printf("  message: %s\n", strings.TrimSpace(doc.Status.Message))
	}

	stages := summarizeWorkflowStages(doc.Status.Nodes)
	for _, stage := range []string{"host_driver", "base_image", "base_test", "app_image", "app_test"} {
		node, ok := stages[stage]
		if !ok {
			continue
		}
		phase := firstNonEmpty(node.Phase, "Unknown")
		message := strings.TrimSpace(node.Message)
		if message != "" {
			fmt.Printf("  %-11s %s - %s\n", stage, phase, message)
		} else {
			fmt.Printf("  %-11s %s\n", stage, phase)
		}
	}
}

func summarizeWorkflowStages(nodes map[string]workflowNode) map[string]workflowNode {
	out := map[string]workflowNode{}
	for _, node := range nodes {
		stage := workflowNodeStage(node)
		if stage == "" {
			continue
		}
		old, ok := out[stage]
		if !ok || workflowNodeScore(node) >= workflowNodeScore(old) {
			out[stage] = node
		}
	}
	return out
}

func workflowNodeStage(node workflowNode) string {
	text := strings.ToLower(strings.Join([]string{node.DisplayName, node.TemplateName}, " "))
	switch {
	case strings.Contains(text, "host-driver"):
		return "host_driver"
	case strings.Contains(text, "base-image"):
		return "base_image"
	case strings.Contains(text, "base-test"):
		return "base_test"
	case strings.Contains(text, "app-image"):
		return "app_image"
	case strings.Contains(text, "app-test"):
		return "app_test"
	default:
		return ""
	}
}

func workflowNodeScore(node workflowNode) int {
	score := 0
	if strings.EqualFold(node.Type, "Pod") {
		score += 10
	}
	switch strings.TrimSpace(node.Phase) {
	case "Failed", "Error":
		score += 8
	case "Succeeded":
		score += 6
	case "Running":
		score += 4
	case "Pending":
		score += 2
	}
	if strings.TrimSpace(node.FinishedAt) != "" {
		score += 1
	}
	return score
}
