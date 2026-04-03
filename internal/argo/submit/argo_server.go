package submit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"sigs.k8s.io/yaml"
)

type ArgoServerClient struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func (c *ArgoServerClient) SubmitWorkflowYAML(ctx context.Context, namespace string, workflowYAML []byte) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("nil client")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return nil, fmt.Errorf("argo server url is required")
	}
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	jsonBody, err := yaml.YAMLToJSON(workflowYAML)
	if err != nil {
		return nil, fmt.Errorf("yaml to json: %w", err)
	}

	u := fmt.Sprintf("%s/api/v1/workflows/%s", base, namespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(c.Token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.Token))
	}

	hc := c.Client
	if hc == nil {
		hc = http.DefaultClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("argo server response %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return b, nil
}
