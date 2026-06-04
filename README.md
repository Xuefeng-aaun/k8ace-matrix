# K8Ace Matrix

K8Ace Matrix 是一套面向算力平台的镜像生产产线。当前版本已经收敛为一个更务实的半自动系统：Dockerfile 由人工或 AI 编写并维护，`images-matrix.yaml` 负责登记可生产目标，`batch-plan*.tsv` 负责声明本轮要生产什么，`matrix-ci` 负责把计划渲染成 Argo Workflow，最后由 Argo 调度 Kaniko 构建镜像并推送到服务器本地 Registry。

一句话理解：

```text
Dockerfile + images-matrix.yaml + batch-plan.tsv
        -> matrix-ci
        -> Argo Workflow
        -> Kaniko 构建
        -> 本地 Registry 保存镜像产物
```

当前可运行版本聚焦 NVIDIA CUDA 生态，已经验证过 `cuda12.4-devel-ubuntu22.04` base image 以及三个 app image：

- `comfyui 0.22.0`
- `llama_factory 0.9.0`
- `stable_diffusion 3.5`

## 当前定位

这个项目现在不是“全自动 Dockerfile 生成器”，也不是“一次性展开所有矩阵组合”的魔法工具。

当前职责：

- 维护一组经过人工确认的 Dockerfile。
- 维护一份可读、可审查的生产矩阵 `images-matrix.yaml`。
- 用 TSV batch plan 声明本轮生产目标。
- 自动完成 context 打包、MinIO 上传、Argo 任务创建、Kaniko 构建和本地 Registry 入库。
- 提供轻量冒烟测试，证明镜像至少具备基础可用性。

当前不负责：

- 不自动从矩阵创作复杂应用 Dockerfile。
- 不保证所有应用都能用同一套模板生成。
- 不做全量异构硬件生产。
- 不默认把产物推送到 Docker Hub。

## 目录结构

```text
.
├── Makefile
├── README.md
├── images-matrix.yaml
├── batch-plan.template.tsv
├── batch-plan.nvidia-cuda.base12.4.tsv
├── batch-plan.nvidia-cuda.practical-3apps.tsv
├── cmd/matrix-ci/
├── internal/
│   ├── matrix/
│   ├── pipeline/
│   └── argo/
├── dockerfiles/
│   ├── base_image/nvidia/cuda124_base/cuda12.4-devel-ubuntu22.04/Dockerfile
│   └── app_image/nvidia/
│       ├── comfyui/0.22.0/comfyui-service-cuda124/Dockerfile
│       ├── llama_factory/0.9.0/llama-factory-service-cuda124/Dockerfile
│       └── stable_diffusion/3.5/stable-diffusion-runtime-cuda124/Dockerfile
├── hack/
│   ├── package_context.sh
│   ├── test/smoke.sh
│   └── local-registry/
├── docs/
└── testdata/
```

主要模块：

- `cmd/matrix-ci`: CLI 入口，提供 `batch`、`render`、`scaffold` 等命令。
- `internal/matrix`: 读取并解析 `images-matrix.yaml`。
- `internal/pipeline`: 根据矩阵和 batch plan 推导构建任务。
- `internal/argo/render`: 把任务渲染成 Argo WorkflowTemplate / Workflow YAML。
- `dockerfiles/base_image`: 共享基础镜像 Dockerfile。
- `dockerfiles/app_image`: 应用镜像 Dockerfile。
- `hack/package_context.sh`: 把仓库打包成 Kaniko 可读取的 `context.tar.gz`。
- `hack/test/smoke.sh`: app image 冒烟测试入口。
- `hack/local-registry`: 本地 Registry 和轻量 Web UI。

## 运行前准备

服务器需要具备：

```bash
go version
make --version
docker version
kubectl version --client
mc --version
```

还需要以下基础服务已经可用：

- Kubernetes + Argo Workflows
- MinIO，作为 Kaniko 构建上下文存储
- 本地 Docker Registry，作为最终镜像产物仓库
- 服务器网络代理或国内镜像链路，用于拉取上游镜像和 pip/apt 依赖

当前服务器默认路径：

```text
/home/xuefeng/newimage/images
```

## 核心配置

### images-matrix.yaml

`images-matrix.yaml` 是人工维护的生产资产。它负责登记：

- 当前支持哪些 base image。
- 当前支持哪些 app image。
- app 依赖哪个精确的 `base_ref`。
- Argo / Kaniko / MinIO / Registry 等产线参数。

当前本地 Registry 前缀：

```yaml
registry_prefix: "172.20.47.182:5000/k8ace"
```

当前 CUDA 12.4 base 使用 DaoCloud public-image-mirror 作为上游来源：

```yaml
base_image_matrix:
  cuda124_base:
    source: "m.daocloud.io/docker.io/nvidia/cuda"
    variants:
      - tag_suffix: "cuda12.4-devel-ubuntu22.04"
        upstream_tag: "12.4.1-devel-ubuntu22.04"
```

这表示 base Dockerfile 的父镜像会被渲染为：

```text
m.daocloud.io/docker.io/nvidia/cuda:12.4.1-devel-ubuntu22.04
```

app 与 base 的关系在 `application_matrix` 中登记。例如：

```yaml
comfyui:
  type: "service"
  versions:
    "0.22.0":
      runtimes:
        cuda:
          cuda-124:
            name: "comfyui-service-cuda124"
            base_ref: "cuda124_base"
            hardware: ["nvidia"]
```

这说明 `comfyui 0.22.0 / comfyui-service-cuda124` 依赖 `cuda124_base`。

### batch plan

`batch-plan*.tsv` 是使用者最常改的文件。它只回答一个问题：本轮要生产什么。

格式：

```text
hardware app_name app_version variant stages
```

字段含义：

- `hardware`: 硬件生态，当前 demo 使用 `nvidia`。
- `app_name`: 应用名，必须能在 `images-matrix.yaml` 中找到。
- `app_version`: 应用版本，不建议使用 `latest`。
- `variant`: 应用变体，必须和矩阵中的 `name` 完全一致。
- `stages`: 本轮生产意图，当前推荐 `base` 或 `app`。

示例：

```tsv
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 base
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
nvidia llama_factory 0.9.0 llama-factory-service-cuda124 app
nvidia stable_diffusion 3.5 stable-diffusion-runtime-cuda124 app
```

当前 stage 展开规则：

```text
base -> base_image -> base_test
app  -> app_image  -> app_test
```

注意：`app` 不再自动携带 `base`。如果 app 依赖的 base 还没有生产过，需要先显式跑一行 `base`。

当前还有一个设计债务：`base` 行仍然需要填写 `app_name / app_version / variant`。原因是程序现在仍通过 app variant 反查 `base_ref`，从而知道要生产哪一个 base。后续可以升级为更直观的 base plan 语法，例如直接声明 `cuda124_base`。

## TSV 与矩阵的关系

以这一行为例：

```tsv
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
```

程序会按下面的路径查找：

```text
TSV hardware = nvidia
TSV app_name = comfyui
TSV app_version = 0.22.0
TSV variant = comfyui-service-cuda124
        ↓
images-matrix.yaml / application_matrix
        ↓
找到 base_ref = cuda124_base
        ↓
images-matrix.yaml / base_image_matrix / cuda124_base
        ↓
找到 source + upstream_tag + tag_suffix
        ↓
生成 Dockerfile 路径、目标镜像名和 Kaniko build args
```

最终会得到：

```text
app Dockerfile:
dockerfiles/app_image/nvidia/comfyui/0.22.0/comfyui-service-cuda124/Dockerfile

app BASE_IMAGE:
172.20.47.182:5000/k8ace/cuda124_base-cuda12.4-devel-ubuntu22.04

app destination:
172.20.47.182:5000/k8ace/comfyui0.22.0-nvidia-comfyui-service-cuda124-dev
```

一句话：TSV 负责“点菜”，矩阵负责“解释菜名”，Dockerfile 负责“真正做菜”。

## Makefile 常用命令

查看帮助：

```bash
make help
```

编译 CLI：

```bash
make build
```

运行测试：

```bash
make test
```

预演生产计划：

```bash
make preview PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
```

`preview` 会：

- 编译 `matrix-ci`
- 读取 `PLAN`
- 渲染 WorkflowTemplate / Workflow YAML
- 打印将要执行的 `kubectl` 命令

`preview` 不会：

- 上传 MinIO
- 创建真实 Argo Workflow
- 构建镜像

正式生产：

```bash
make produce PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
```

`produce` 会执行：

```text
go build
-> go test
-> bash hack/package_context.sh
-> mc cp dist/context/context.tar.gz 到 MinIO
-> matrix-ci batch 渲染 Argo YAML
-> kubectl apply WorkflowTemplate
-> kubectl create Workflow
-> 等待 Workflow 结束并打印阶段摘要
```

当前 demo 快捷命令：

```bash
make demo-preview
make demo
```

只生产 CUDA 12.4 base：

```bash
make base-preview
make base-produce
```

## 完整生产流程

当前可运行链路如下：

```text
1. 编写或确认 Dockerfile
2. 在 images-matrix.yaml 登记 app/base 关系
3. 编写 batch-plan.tsv
4. make preview PLAN=...
5. make produce PLAN=...
6. hack/package_context.sh 打包仓库
7. 上传 context.tar.gz 到 MinIO
8. matrix-ci 生成 Argo WorkflowTemplate / Workflow
9. Argo 创建 Kaniko Pod
10. Kaniko 从 MinIO 下载 context
11. Kaniko 按 Dockerfile 构建镜像
12. 镜像推送到本地 Registry
13. Argo 启动 test 阶段做冒烟测试
```

当前 MinIO 只保存构建上下文：

```text
s3://kaniko-contexts/k8ace/context.tar.gz
```

最终镜像不保存在 MinIO，而是保存在本地 Registry：

```text
172.20.47.182:5000/k8ace/...
```

## Dockerfile 编写细则

当前 Dockerfile 的要求已经提高。它不再只是“能安装完依赖”，还必须带上可被产线自动验收的契约。

每个 app Dockerfile 至少要满足：

- 必须基于 `ARG BASE_IMAGE` / `FROM ${BASE_IMAGE}`，让产线可以把 app 构建在已生产的 base image 上。
- 必须复制 `hack/test/smoke.sh` 到 `/opt/k8ace/hack/test/smoke.sh`。
- service 镜像必须提供稳定启动入口，并声明服务契约环境变量。
- runtime 镜像必须声明可 import 的关键包或自定义 runtime 检查命令。
- cli 镜像必须声明可成功返回的版本或帮助命令。
- Python 依赖安装后必须执行 `python3 -m pip check`，提前暴露依赖冲突。
- 对上游依赖范围过宽的项目，要使用 constraints 文件固定关键依赖线，避免未来解析到不兼容新版本。

三类契约示例：

```dockerfile
# service：必须能启动并被 curl 通。
ENV K8ACE_SMOKE_TYPE=service \
    K8ACE_SMOKE_SERVICE_CMD=/opt/k8ace/hack/images/comfyui/entrypoint.sh \
    K8ACE_SMOKE_PORT=8188 \
    K8ACE_SMOKE_HEALTH_PATH=/ \
    K8ACE_SMOKE_TIMEOUT_SECONDS=180

# runtime：必须能 import 关键包。
ENV K8ACE_SMOKE_TYPE=runtime \
    K8ACE_SMOKE_IMPORTS=torch,diffusers,transformers,accelerate

# cli：必须有一个能成功返回的命令。
ENV K8ACE_SMOKE_TYPE=cli \
    K8ACE_SMOKE_CLI_CMD="some-cli --help"
```

LLaMA-Factory 这种 WebUI 应用现在按 service 镜像处理。它的 v0.9.0 依赖范围较宽，Dockerfile 中使用 `/tmp/llamafactory-constraints.txt` 固定 Gradio 4.x 稳定线，并用 `pip check` 检查依赖一致性。这样可以避免再次出现 Gradio 版本解析过新导致 WebUI 启动失败的问题。

## 冒烟测试

当前冒烟测试不做真实模型推理，但会验证镜像是否能按交付类型最低可用。

`base_test` 当前检查：

```bash
python3 --version
pip --version || pip3 --version
```

`app_test` 会执行镜像内的：

```bash
/opt/k8ace/hack/test/smoke.sh
```

当前实际渲染为 L3：

```bash
bash /opt/k8ace/hack/test/smoke.sh L3 comfyui nvidia service
bash /opt/k8ace/hack/test/smoke.sh L3 llama_factory nvidia service
bash /opt/k8ace/hack/test/smoke.sh L3 stable_diffusion nvidia runtime
```

当前三类规则：

- `service`: 启动服务，等待端口可用，`curl` 健康地址成功。
- `runtime`: 直接 import 关键 Python 包成功。
- `cli`: 执行版本或帮助命令并成功返回。

当前三个 app 的规则：

- `comfyui`: service，启动 8188 并 `curl /`。
- `llama_factory`: service，启动 7860 并 `curl /`。
- `stable_diffusion`: runtime，import `torch / diffusers / transformers / accelerate`，并导入 `StableDiffusion3Pipeline`。

L2 硬件级测试目前仍保留为后续扩展，但 demo 阶段优先验证 Dockerfile -> Kaniko -> Registry -> app_test 主链路。

## 本地 Registry 与 Web UI

本地 Registry 配置目录：

```bash
cd /home/xuefeng/newimage/images/hack/local-registry
```

启动：

```bash
docker compose up -d
docker compose ps
```

Registry API：

```bash
curl http://127.0.0.1:5000/v2/_catalog
```

Registry Web UI：

```text
http://172.20.47.182:8088
```

如果要用 SSH 转发：

```powershell
ssh -i "E:\rsa\id_rsa_2048" -L 8088:127.0.0.1:8088 xuefeng@172.20.47.182
```

然后本地打开：

```text
http://127.0.0.1:8088
```

Web UI 可以查看仓库、tag 和 digest，也可以删除 tag。删除 tag 后不一定立刻释放磁盘空间，真正释放空间需要执行 garbage collect：

```bash
cd /home/xuefeng/newimage/images/hack/local-registry

docker compose stop registry-ui
docker compose stop registry
docker compose run --rm registry registry garbage-collect /etc/docker/registry/config.yml
docker compose up -d
```

## docker run 使用示例

当前三个 app 镜像 tag 都是 `latest`。

### ComfyUI

ComfyUI 是 service image，可以直接启动 WebUI：

```bash
docker run --rm -it \
  --gpus all \
  -p 8188:8188 \
  --name k8ace-comfyui \
  172.20.47.182:5000/k8ace/comfyui0.22.0-nvidia-comfyui-service-cuda124-dev:latest
```

访问：

```text
http://172.20.47.182:8188
```

如果 `8188` 被占用，可以换宿主机端口：

```bash
docker run --rm -it \
  --gpus all \
  -p 8190:8188 \
  --name k8ace-comfyui-new \
  172.20.47.182:5000/k8ace/comfyui0.22.0-nvidia-comfyui-service-cuda124-dev:latest
```

访问：

```text
http://172.20.47.182:8190
```

### LLaMA-Factory

LLaMA-Factory 当前按 service image 交付，默认启动 WebUI：

```bash
docker run --rm -it \
  --gpus all \
  -p 7860:7860 \
  --name k8ace-llamafactory \
  172.20.47.182:5000/k8ace/llama_factory0.9.0-nvidia-llama-factory-service-cuda124-dev:latest
```

访问：

```text
http://172.20.47.182:7860
```

### Stable Diffusion Runtime

当前 Stable Diffusion 镜像是 runtime image，不是 WebUI service。它默认进入 Python 环境。

启动：

```bash
docker run --rm -it \
  --gpus all \
  --name k8ace-stable-diffusion \
  172.20.47.182:5000/k8ace/stable_diffusion3.5-nvidia-stable-diffusion-runtime-cuda124-dev:latest
```

做 import 检查：

```bash
docker run --rm -it \
  --gpus all \
  172.20.47.182:5000/k8ace/stable_diffusion3.5-nvidia-stable-diffusion-runtime-cuda124-dev:latest \
  -c "import diffusers; from diffusers import StableDiffusion3Pipeline; print('stable diffusion runtime ok')"
```

## 查看 Argo 状态

查看 Workflow：

```bash
kubectl -n default get wf
```

查看某个 Workflow 的 Pod：

```bash
WF=<workflow-name>
kubectl -n default get pods -l workflows.argoproj.io/workflow=$WF
```

查看日志：

```bash
POD=<pod-name>
kubectl -n default logs $POD -c main --tail=120 -f
```

Argo Web UI 当前 NodePort：

```text
https://172.20.47.182:32097
```

如果用 SSH 转发：

```powershell
ssh -i "E:\rsa\id_rsa_2048" -L 2746:127.0.0.1:32097 xuefeng@172.20.47.182
```

本地访问：

```text
https://127.0.0.1:2746
```

Argo 登录 token 需要带 `Bearer` 前缀：

```bash
kubectl -n argo create token argo-server --duration=24h | sed 's/^/Bearer /'
```

## 常见问题

### 1. docker run 提示端口被占用

报错示例：

```text
Bind for 0.0.0.0:8188 failed: port is already allocated
```

查看占用：

```bash
sudo ss -lntp | grep ':8188'
docker ps --format 'table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Ports}}'
```

解决方式：

- 停掉旧容器。
- 或者换一个宿主机端口，例如 `-p 8190:8188`。

### 2. Kaniko 长时间停在 Taking snapshot

这通常不是 Argo 调度失败，也不是 CPU 没用上，而是 Kaniko 在扫描文件系统、计算层差异。

常见原因：

- Dockerfile 中 pip install 后又执行多个小 `RUN/COPY/chmod`。
- 大量 Python 包导致文件数量很多。
- 多个 Kaniko Pod 同时 snapshot，互相抢磁盘 IO。

优化方向：

- 合并 Dockerfile 中的 `RUN`。
- 把小文件 `COPY` 和 `chmod` 尽量放到大安装步骤之前。
- 降低 app image 并发。
- 后续可评估 Kaniko `--snapshot-mode=redo`。

### 3. Argo token not valid

Argo UI 登录时 token 必须带 `Bearer` 前缀：

```bash
kubectl -n argo create token argo-server --duration=24h | sed 's/^/Bearer /'
```

### 4. DaoCloud / Docker Hub / 代理链路

当前 base image 使用 DaoCloud 显式前缀：

```text
m.daocloud.io/docker.io/nvidia/cuda:12.4.1-devel-ubuntu22.04
```

服务器 Mihomo 中已将 DaoCloud 相关域名设置为 `DIRECT`，避免走被限速的代理节点：

```text
m.daocloud.io
daocloud.io
image-mirror.r2.daocloud.vip
daocloud.vip
```

其他外部链路仍可通过代理访问，例如 GitHub、PyPI、Docker Hub token 等。

## 当前已知取舍

- `host_driver` 暂时屏蔽，避免 demo 被 Kubernetes GPU device plugin 等基础设施问题卡住。
- `base` 和 `app` 分开写 plan，避免多个 app 重复构建共享 base。
- `base` 行仍需借助 app variant 反查 `base_ref`，这是当前设计债务。
- 冒烟测试当前做到 L3：service 必须启动并 curl，runtime 必须 import，cli 必须命令返回；暂不做真实模型推理。
- Stable Diffusion 当前是 runtime image，不是 WebUI image。
- Registry Web UI 可删除 tag，但释放空间仍需 garbage collect。

## 推荐日常流程

```text
1. 写好或修改 Dockerfile
2. 在 images-matrix.yaml 登记 app/base 关系
3. 写 batch-plan.tsv
4. make preview PLAN=xxx.tsv
5. make produce PLAN=xxx.tsv
6. 在 Argo 查看构建状态
7. 在 Registry Web UI 查看产物
8. 用 docker run 做落地测试
```

当前三应用 demo：

```bash
cd /home/xuefeng/newimage/images
make preview PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
make produce PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
```
