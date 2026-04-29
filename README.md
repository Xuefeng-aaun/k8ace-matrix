# Images Matrix CI

本仓库当前按单一路线维护：`images-matrix.yaml` 作为镜像矩阵配置中心，`matrix-ci` 根据矩阵生成 Dockerfile 和 Argo Workflow，Argo Workflows 调度 Kaniko 构建镜像，MinIO/S3 提供 Kaniko 构建上下文，本地 Docker Registry 保存最终镜像产物。

当前目标不是做一个完全通用的平台，而是先把当前服务器环境中的自动化镜像产线稳定跑通，并把每一步拆开到可以人工审查、人工复现、后续再自动化的程度。

## 当前固定链路

- 项目目录：`/home/xuefeng/images`
- 主配置文件：`images-matrix.yaml`
- CLI 二进制：`./bin/matrix-ci`
- 构建上下文：`s3://kaniko-contexts/k8ace/context.tar.gz`
- MinIO endpoint：`http://172.20.47.182:9000`
- 本地镜像仓库：`172.20.47.182:5000/k8ace`
- Argo namespace：`default`
- Argo service account：`argo-workflow`
- Kaniko executor：`gcr.io/kaniko-project/executor:v1.23.2`
- 本地 registry 是 HTTP 服务，因此渲染出的 Kaniko 参数会包含 `--insecure-registry=172.20.47.182:5000`
- 当前不使用 Docker Hub 推送，也不依赖 Docker Hub 作为最终产物仓库。

## 项目整体流程

```text
images-matrix.yaml
  -> matrix-ci scaffold dockerfiles
  -> 生成 dockerfiles/base_image/.../Dockerfile
  -> 生成 dockerfiles/app_image/.../Dockerfile
  -> hack/package_context.sh 打包构建上下文
  -> 上传 context.tar.gz 到 MinIO
  -> matrix-ci render 生成 Argo WorkflowTemplate
  -> kubectl apply / create Workflow
  -> Argo 调度 Kaniko Pod
  -> Kaniko 从 MinIO 下载 context.tar.gz
  -> Kaniko 按 --dockerfile 找到对应 Dockerfile
  -> 构建 base image
  -> 构建 app image
  -> 推送到本地 Registry
```

一句话理解：

```text
matrix 负责描述要构建什么，scaffold 负责生成 Dockerfile，context 负责打包构建材料，render 负责生成 Argo 任务，Argo/Kaniko 负责真正构建镜像。
```

## 本次重要改动

### 1. 支持精确选择 base image variant

原先 app variant 里通常只写：

```yaml
base_ref: "cuda_base"
```

这只能表示“我要 CUDA 这个大类”，但不能说明到底要用：

```text
cuda12.4-devel-ubuntu22.04
cuda12.2-devel-ubuntu22.04
cuda11.8-devel-ubuntu20.04
```

因此本次增加了 `base_tag_suffix` 选择能力。它可以出现在两处。

第一种是命令行临时覆盖：

```bash
--base-tag-suffix=cuda12.4-devel-ubuntu22.04
```

第二种是未来写进 `images-matrix.yaml` 的 app variant：

```yaml
variants:
  - name: "sd-webui-cuda"
    base_ref: "cuda_base"
    base_tag_suffix: "cuda12.4-devel-ubuntu22.04"
    hardware: ["nvidia"]
```

推荐长期把确定的 app/base 配套关系沉淀进 `images-matrix.yaml`，命令行参数更适合临时测试。

### 2. base variant 选择顺序

现在 base image variant 的选择逻辑是：

1. 如果命令行传了 `--base-tag-suffix`，优先使用命令行指定值。
2. 如果 app variant 里写了 `base_tag_suffix`，使用 YAML 中的指定值。
3. 如果没有显式指定，优先选择 `status: recommended`。
4. 如果没有 `recommended`，选择 `status: stable`。
5. 如果都没有，选择第一个兼容当前 hardware 的 variant。
6. 如果指定了不存在或不兼容的 `base_tag_suffix`，命令直接失败，不再静默 fallback 到错误 base。

这能避免 app image 在看似生成成功的情况下，实际使用了错误 CUDA/CANN/Python/Ubuntu 组合。

### 3. `status` 字段开始具备实际意义

`status` 原先更接近备注字段。现在它参与默认选择：

```yaml
status: "recommended"
status: "stable"
status: "legacy"
```

当前策略：

- `recommended`：默认优先选择，适合作为当前主推版本。
- `stable`：次优选择，适合稳定但非最新的版本。
- `legacy`：保留兼容旧环境，不会在默认策略中优先选中。

### 4. scaffold/render/submit 都支持 `--base-tag-suffix`

以下入口都已经支持同一个参数：

```bash
./bin/matrix-ci scaffold dockerfiles --base-tag-suffix=...
./bin/matrix-ci render --base-tag-suffix=...
./bin/matrix-ci submit --base-tag-suffix=...
```

这保证了手动生成 Dockerfile、渲染 Workflow、直接提交 Workflow 时都能使用同一套 base 选择逻辑。

## base image 与 app image 的关系

每一个 app image 理论上都应该有一份配套的 base image。

以 `sd_webui + nvidia + cuda12.4` 为例：

```text
base image Dockerfile:
dockerfiles/base_image/nvidia/cuda_base/cuda12.4-devel-ubuntu22.04/Dockerfile

app image Dockerfile:
dockerfiles/app_image/nvidia/sd_webui/1.10.0/sd-webui-cuda/Dockerfile
```

app Dockerfile 不是通过文件路径直接读取 base Dockerfile，而是通过镜像产物连接。

base image 阶段会产出：

```text
172.20.47.182:5000/k8ace/cuda_base-cuda12.4-devel-ubuntu22.04
```

app image 阶段的 Dockerfile 里通常是：

```dockerfile
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
```

Argo/Kaniko 构建 app image 时，会把 base image 的产物地址作为构建参数传进去：

```text
BASE_IMAGE=172.20.47.182:5000/k8ace/cuda_base-cuda12.4-devel-ubuntu22.04
```

所以真正的依赖关系是：

```text
app variant -> base_ref -> base_tag_suffix/status -> base variant -> base image 产物 -> app image 的 BASE_IMAGE
```

## 从零开始的人工审查流程

以下流程适合提交 GitHub 前进行人工复现和审查。示例使用 `sd_webui 1.10.0 / sd-webui-cuda / nvidia / cuda12.4`。

### 1. 进入项目目录

```bash
cd /home/xuefeng/images
```

确认当前分支：

```bash
git branch --show-current
```

如果是提交审查分支，当前应为：

```text
review-submit
```

查看是否有未提交修改：

```bash
git status --short
```

### 2. 编译 matrix-ci

```bash
go build -buildvcs=false -o ./bin/matrix-ci ./cmd/matrix-ci
```

如果服务器上的目录不是完整 Git checkout，Go 可能会因为 VCS stamping 报错，所以这里固定使用：

```bash
-buildvcs=false
```

### 3. 运行测试

```bash
go test ./...
```

如果只是快速验证核心逻辑，也可以先跑：

```bash
go test ./internal/matrix ./internal/pipeline ./internal/scaffold ./internal/argo/render
```

### 4. 手动生成 base image Dockerfile

```bash
./bin/matrix-ci scaffold dockerfiles \
  --matrix=./images-matrix.yaml \
  --hardware=nvidia \
  --app-name=sd_webui \
  --app-version=1.10.0 \
  --variant=sd-webui-cuda \
  --base-tag-suffix=cuda12.4-devel-ubuntu22.04 \
  --stage=base_image
```

生成位置：

```text
dockerfiles/base_image/nvidia/cuda_base/cuda12.4-devel-ubuntu22.04/Dockerfile
```

查看内容：

```bash
sed -n '1,120p' dockerfiles/base_image/nvidia/cuda_base/cuda12.4-devel-ubuntu22.04/Dockerfile
```

这份 Dockerfile 负责系统层和基础依赖，通常从上游基础镜像开始：

```dockerfile
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
```

base image 构建时的 `BASE_IMAGE` 通常来自 `base_image_matrix` 中的 `source + upstream_tag`，例如：

```text
nvidia/cuda:12.4.1-devel-ubuntu22.04
```

### 5. 手动生成 app image Dockerfile

```bash
./bin/matrix-ci scaffold dockerfiles \
  --matrix=./images-matrix.yaml \
  --hardware=nvidia \
  --app-name=sd_webui \
  --app-version=1.10.0 \
  --variant=sd-webui-cuda \
  --base-tag-suffix=cuda12.4-devel-ubuntu22.04 \
  --stage=app_image
```

生成位置：

```text
dockerfiles/app_image/nvidia/sd_webui/1.10.0/sd-webui-cuda/Dockerfile
```

查看内容：

```bash
sed -n '1,160p' dockerfiles/app_image/nvidia/sd_webui/1.10.0/sd-webui-cuda/Dockerfile
```

app image Dockerfile 的核心结构是：

```dockerfile
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
```

它表示 app image 会基于前一步构建出来的 base image 继续安装应用层依赖。

### 6. 一次性生成 base/app 两份 Dockerfile

如果已经确认参数正确，可以一次性生成配套 Dockerfile：

```bash
./bin/matrix-ci scaffold dockerfiles \
  --matrix=./images-matrix.yaml \
  --hardware=nvidia \
  --app-name=sd_webui \
  --app-version=1.10.0 \
  --variant=sd-webui-cuda \
  --base-tag-suffix=cuda12.4-devel-ubuntu22.04 \
  --stage=base_image \
  --stage=app_image
```

如果只是学习或临时验证，不想污染项目目录，建议在 `/tmp` 下执行：

```bash
mkdir -p /tmp/matrix-scaffold-test
cd /tmp/matrix-scaffold-test

/home/xuefeng/images/bin/matrix-ci scaffold dockerfiles \
  --matrix=/home/xuefeng/images/images-matrix.yaml \
  --hardware=nvidia \
  --app-name=sd_webui \
  --app-version=1.10.0 \
  --variant=sd-webui-cuda \
  --base-tag-suffix=cuda12.4-devel-ubuntu22.04 \
  --stage=base_image \
  --stage=app_image

find ./dockerfiles -type f -name Dockerfile
```

### 7. 打包 Kaniko 构建上下文

回到项目目录：

```bash
cd /home/xuefeng/images
bash hack/package_context.sh
```

默认输出：

```text
dist/context/context.tar.gz
```

构建上下文不是某一份 Dockerfile，而是 Kaniko 构建时需要的材料包。它至少要包含：

```text
dockerfiles/
hack/
images-matrix.yaml
其他构建时需要的脚本或配置
```

检查 context 中是否包含刚刚生成的 Dockerfile：

```bash
tar -tzf dist/context/context.tar.gz | grep 'dockerfiles/base_image/nvidia/cuda_base/cuda12.4-devel-ubuntu22.04/Dockerfile'
tar -tzf dist/context/context.tar.gz | grep 'dockerfiles/app_image/nvidia/sd_webui/1.10.0/sd-webui-cuda/Dockerfile'
```

如果这里查不到，后面 Kaniko 会报类似错误：

```text
error resolving dockerfile path: please provide a valid path to a Dockerfile within the build context with --dockerfile
```

### 8. 上传 context 到 MinIO

如果服务器已经配置好 `mc` alias，执行：

```bash
mc cp dist/context/context.tar.gz local/kaniko-contexts/k8ace/context.tar.gz
```

最终对象地址需要对应：

```text
s3://kaniko-contexts/k8ace/context.tar.gz
```

`images-matrix.yaml` 中当前 MinIO 相关配置位于：

```yaml
ci_cd:
  argo_workflows:
    build_context:
      default: "s3://kaniko-contexts/k8ace/context.tar.gz"
      env:
        S3_ENDPOINT: "http://172.20.47.182:9000"
      secret_name: "kaniko-context-credentials"
```

确认 Secret 存在：

```bash
kubectl -n default get secret kaniko-context-credentials
```

### 9. 确认本地 Registry 可用

本地 registry 地址：

```text
172.20.47.182:5000
```

检查接口：

```bash
curl -sS http://127.0.0.1:5000/v2/
```

正常情况下会返回：

```json
{}
```

查看已有仓库：

```bash
curl -sS http://127.0.0.1:5000/v2/_catalog
```

### 10. 渲染 WorkflowTemplate

```bash
cd /home/xuefeng/images

./bin/matrix-ci render \
  --matrix=./images-matrix.yaml \
  --hardware=nvidia \
  --app-name=sd_webui \
  --app-version=1.10.0 \
  --variant=sd-webui-cuda \
  --base-tag-suffix=cuda12.4-devel-ubuntu22.04 \
  --out-dir=./dist/argo
```

渲染产物位置：

```text
dist/argo/workflowtemplates/k8ace-matrix-nvidia-sd-webui-1-10-0-sd-webui-cuda.yaml
```

检查 WorkflowTemplate 中是否指向 MinIO context：

```bash
grep -n -- '--context=\|--dockerfile=\|--destination=\|BASE_IMAGE' \
  dist/argo/workflowtemplates/k8ace-matrix-nvidia-sd-webui-1-10-0-sd-webui-cuda.yaml
```

重点确认三件事：

```text
--context=s3://kaniko-contexts/k8ace/context.tar.gz
--dockerfile=dockerfiles/base_image/... 或 dockerfiles/app_image/...
--destination=172.20.47.182:5000/k8ace/...
```

### 11. 提交 WorkflowTemplate

```bash
kubectl apply -f ./dist/argo/workflowtemplates/k8ace-matrix-nvidia-sd-webui-1-10-0-sd-webui-cuda.yaml
```

### 12. 创建一次 Workflow

```bash
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

### 13. 查看 Workflow 状态

获取最新 Workflow 名称：

```bash
WF=$(kubectl -n default get wf --sort-by=.metadata.creationTimestamp -o name | grep k8ace-matrix-nvidia-sd-webui | tail -1 | cut -d/ -f2)
echo "$WF"
```

查看 Workflow：

```bash
kubectl -n default get wf "$WF"
```

查看 Pod：

```bash
kubectl -n default get pods -l workflows.argoproj.io/workflow="$WF" -o wide
```

查看日志：

```bash
POD=$(kubectl -n default get pods -l workflows.argoproj.io/workflow="$WF" -o name | head -1 | cut -d/ -f2)
kubectl -n default logs "$POD" -c main --tail=160 -f
```

如果有多个阶段，可以按 Pod 名称选择对应阶段，例如 `base-image` 或 `app-image`。

### 14. 查看本地 Registry 产物

查看 catalog：

```bash
curl -sS http://127.0.0.1:5000/v2/_catalog
```

查看 sd_webui app image tags：

```bash
curl -sS http://127.0.0.1:5000/v2/k8ace/sd_webui1.10.0-nvidia-sd-webui-cuda-dev/tags/list
```

如果镜像仓库名发生变化，以 render 出来的 `--destination=` 为准。

## 常见问题

### 1. `--base-tag-suffix` 可以不填吗

可以。不填时会按以下顺序自动选择：

```text
recommended -> stable -> 第一个兼容当前 hardware 的 variant
```

但对于需要严格控制 CUDA/CANN/Python/Ubuntu 版本的 app，建议显式写入 `images-matrix.yaml` 的 app variant，或在命令行临时传入。

### 2. `--base-tag-suffix` 填错会怎样

会直接失败。这样做是为了防止自动选择错误 base，导致后续镜像构建出错但原因不明显。

### 3. 构建上下文知道要找哪份 Dockerfile 吗

不知道。构建上下文只是材料包。

真正指定 Dockerfile 路径的是 Argo/Kaniko 参数：

```text
--context=s3://kaniko-contexts/k8ace/context.tar.gz
--dockerfile=dockerfiles/app_image/nvidia/sd_webui/1.10.0/sd-webui-cuda/Dockerfile
```

Kaniko 会下载 context，然后在 context 内按 `--dockerfile` 指定路径找文件。

### 4. app Dockerfile 会直接读取 base Dockerfile 吗

不会。app Dockerfile 通过镜像产物依赖 base image：

```dockerfile
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
```

base image 先被构建并推送到本地 Registry，app image 再以这个 base image 的地址作为 `BASE_IMAGE`。

### 5. 为什么需要 MinIO

Kaniko Pod 在 Kubernetes 集群里运行，它不能天然读取你服务器本地 `/home/xuefeng/images` 目录。MinIO/S3 的作用是把项目构建材料打包成 `context.tar.gz`，让 Kaniko Pod 可以通过 S3 地址下载。

### 6. 为什么推送到本地 Registry

当前服务器网络环境对 Docker Hub 推送链路不稳定。先推送到本地 Registry 可以把“产线是否能构建成功”和“公网仓库上传是否稳定”拆开，降低排错难度。

## 提交到 GitHub 前建议检查

只提交源码、配置和必要文档，不提交缓存、产物和临时文件。

推荐检查：

```bash
git status --short
```

推荐提交核心文件：

```bash
git add \
  README.md \
  cmd/matrix-ci/render.go \
  cmd/matrix-ci/scaffold.go \
  cmd/matrix-ci/submit.go \
  internal/matrix/types.go \
  internal/pipeline/plan.go \
  internal/pipeline/unit.go \
  internal/pipeline/unit_test.go
```

不要直接 `git add .`，除非你已经确认没有临时文件。

提交：

```bash
git commit -m "Document matrix Dockerfile workflow"
git push origin review-submit
```

如果前一次代码修改还没提交，可以把 README 和代码一起提交：

```bash
git commit -m "Support explicit base image variant selection"
git push origin review-submit
```

## 保留的关键文件

- `images-matrix.yaml`：主配置文件，描述 base image、app image、构建阶段、CI/CD、MinIO、本地 Registry 等信息。
- `cmd/`：`matrix-ci` 命令行入口。
- `internal/matrix/`：读取和解析 `images-matrix.yaml`。
- `internal/pipeline/`：从 matrix 推导 build unit 和 build plan。
- `internal/scaffold/`：根据 build unit 生成 Dockerfile。
- `internal/argo/render/`：根据 build plan 渲染 Argo Workflow/WorkflowTemplate。
- `dockerfiles/`：实际镜像构建入口。
- `hack/package_context.sh`：生成 Kaniko 使用的 `context.tar.gz`。
- `hack/local-registry/docker-compose.yaml`：本地 Docker Registry 配置。
- `CODE_WIKI.md`：项目结构和模块说明。