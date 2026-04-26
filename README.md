# Images Matrix CI

本仓库当前按单一路线维护：Argo Workflows + Kaniko 构建镜像，MinIO/S3 提供构建上下文，本地 Docker Registry 保存产物。目标不是做通用模板，而是先把当前服务器环境稳定跑通。

## 当前固定链路

- 构建上下文：`s3://kaniko-contexts/k8ace/context.tar.gz`
- MinIO endpoint：`http://172.20.47.182:9000`
- 本地镜像仓库：`172.20.47.182:5000/k8ace`
- 本地 registry 为 HTTP，因此渲染出的 Kaniko 参数会包含 `--insecure-registry=172.20.47.182:5000`
- 当前不使用 Docker Hub 推送，也不保留 Docker Hub 镜像站配置。

## 一次完整构建流程

### 1. 打包构建上下文

```bash
cd /home/xuefeng/images
bash hack/package_context.sh
```

默认生成：`dist/context/context.tar.gz`。

### 2. 上传 context 到 MinIO

如果已经安装并配置 `mc`，执行：

```bash
mc cp dist/context/context.tar.gz local/kaniko-contexts/k8ace/context.tar.gz
```

如果 bucket 或 alias 名称不同，以服务器当前 MinIO 配置为准；关键是最终对象地址要对应 `s3://kaniko-contexts/k8ace/context.tar.gz`。

### 3. 确认 Secret 存在

```bash
kubectl -n default get secret kaniko-context-credentials
kubectl -n default get secret registry-credentials
```

`kaniko-context-credentials` 用于读取 MinIO/S3 context；`registry-credentials` 当前保留给 Kaniko docker config 挂载，虽然本地 registry 不强制鉴权。

### 4. 编译 CLI

```bash
cd /home/xuefeng/images
go build -buildvcs=false -o ./bin/matrix-ci ./cmd/matrix-ci
```

### 5. 渲染 WorkflowTemplate

```bash
cd /home/xuefeng/images
./bin/matrix-ci render \
  --matrix ./images-matrix.yaml \
  --hardware nvidia \
  --app-name sd_webui \
  --app-version 1.10.0 \
  --variant sd-webui-cuda \
  --out-dir ./dist/argo
```

### 6. 提交 Argo workflow

```bash
cd /home/xuefeng/images
kubectl apply -f ./dist/argo/workflowtemplates/k8ace-matrix-nvidia-sd-webui-1-10-0-sd-webui-cuda.yaml
kubectl -n default create -f - <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8ace-matrix-nvidia-sd-webui-
  namespace: default
spec:
  workflowTemplateRef:
    name: k8ace-matrix-nvidia-sd-webui-1-10-0-sd-webui-cuda
EOF
```

### 7. 查看状态

```bash
WF=$(kubectl -n default get wf --sort-by=.metadata.creationTimestamp -o name | grep k8ace-matrix-nvidia-sd-webui | tail -1 | cut -d/ -f2)
kubectl -n default get wf "$WF"
kubectl -n default get pods -l workflows.argoproj.io/workflow="$WF" -o wide
kubectl -n default logs -l workflows.argoproj.io/workflow="$WF" -c main --tail=120 -f
```

### 8. 查看本地 registry 产物

```bash
curl -sS http://127.0.0.1:5000/v2/_catalog
curl -sS http://127.0.0.1:5000/v2/k8ace/sd_webui1.10.0-nvidia-sd-webui-cuda-dev/tags/list
```

## 保留的关键文件

- `images-matrix.yaml`：唯一主配置，已经内置当前服务器的 MinIO 和本地 registry 参数。
- `hack/package_context.sh`：生成 Kaniko 使用的 `context.tar.gz`。
- `hack/local-registry/docker-compose.yaml`：启动本地 Docker Registry。
- `cmd/`、`internal/`：`matrix-ci` CLI 与 Argo/Kaniko 渲染逻辑。
- `dockerfiles/`：实际镜像构建入口。
