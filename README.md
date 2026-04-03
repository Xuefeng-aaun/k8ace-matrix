# Images Matrix CI（开发者文档）

本仓库用于基于 [images-matrix.yaml](file:///Users/jarvis/Documents/trae_projects/images/images-matrix.yaml) 维护“异构加速卡镜像构建矩阵”，并通过 Go CLI（matrix-ci）生成/提交 Argo Workflows（Kaniko 构建）。

更详细的新增镜像与 CI 扩展规范见：[DEVELOPERS.md](file:///Users/jarvis/Documents/trae_projects/images/DEVELOPERS.md)。

## 快速开始

### 1) 渲染 Argo WorkflowTemplate（推荐）

```bash
go run ./cmd/matrix-ci render \
  --matrix ./images-matrix.yaml \
  --hardware nvidia \
  --app-name sd_webui \
  --app-version 1.10.0 \
  --variant sd-webui-cuda \
  --out-dir dist/argo
```

### 2) 直接提交到 argo-server（可选）

```bash
export MATRIX_CI_ARGO_TOKEN="***"
go run ./cmd/matrix-ci submit \
  --matrix ./images-matrix.yaml \
  --argo-server https://argo.example.com \
  --namespace default \
  --hardware nvidia \
  --app-name sd_webui \
  --app-version 1.10.0 \
  --variant sd-webui-cuda
```

### 3) 生成 Dockerfile（可选）

```bash
go run ./cmd/matrix-ci scaffold dockerfiles \
  --matrix ./images-matrix.yaml \
  --hardware nvidia \
  --app-name pytorch \
  --app-version 2.5.1 \
  --variant pytorch-cuda \
  --stage base_image --stage app_image
```

