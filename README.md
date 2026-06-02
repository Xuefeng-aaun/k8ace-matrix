# k8ace-matrix

`k8ace-matrix` 是一套面向 CUDA 场景的半自动镜像生产仓库。

它现在的定位很明确：

- Dockerfile 由人工或 AI 参考上游资料编写
- `images-matrix.yaml` 负责登记支持范围和运行参数
- `batch plan` 负责选择“这次要生产什么”
- `matrix-ci + Argo + Kaniko + MinIO + 本地 Registry` 负责批量构建、测试和产物落库

它不再追求“从矩阵自动生成所有 Dockerfile”。

## 当前支持

- 硬件：`nvidia`
- 共享基础镜像：`cuda12.4-devel-ubuntu22.04`
- 当前实战应用：
  - `comfyui`
  - `llama_factory`
  - `stable_diffusion`

## 仓库怎么理解

这套仓库可以看成 4 层：

1. Dockerfile 配方层  
   决定镜像里真正装什么、默认怎么启动。

2. 编排配置层  
   由 `images-matrix.yaml` 承担，负责声明当前仓库支持哪些 base、哪些 app、有哪些 stage。

3. 批次计划层  
   由 `batch-plan.*.tsv` 承担，负责决定“这次跑哪几个目标”。

4. 执行层  
   由 `matrix-ci`、Argo、Kaniko、MinIO、本地 Registry 组成，负责把镜像真正做出来。

## 当前主链

当前主链固定为：

```text
写 Dockerfile
-> 写/维护 matrix 记录
-> 写 plan
-> 打包 context
-> 上传 MinIO
-> 渲染 Argo Workflow
-> host_driver 检查 NVIDIA GPU/驱动/CUDA 兼容性
-> Argo 调度 Kaniko
-> 构建 base_image / app_image
-> 执行 smoke 测试
-> 推送到本地 Registry
```

## 目录结构

```text
images-matrix.yaml
batch-plan.nvidia-cuda.base12.4.tsv
batch-plan.nvidia-cuda.practical-3apps.tsv
Makefile

cmd/matrix-ci/
internal/matrix/
internal/pipeline/
internal/argo/
internal/version/

dockerfiles/
  base_image/nvidia/cuda124_base/cuda12.4-devel-ubuntu22.04/
  app_image/nvidia/comfyui/0.22.0/comfyui-service-cuda124/
  app_image/nvidia/llama_factory/0.9.0/llama-factory-cli-cuda124/
  app_image/nvidia/stable_diffusion/3.5/stable-diffusion-runtime-cuda124/

hack/
  package_context.sh
  test/smoke.sh
  images/comfyui/entrypoint.sh
  images/llama-factory/entrypoint.sh
  local-registry/

docs/
  practical-app-dockerfile-sources.md
  application-taxonomy.md
```

## 第一次拿到项目怎么做

### 1. 先确认环境

至少需要：

- Go
- GNU Make
- Docker
- kubectl
- Argo Workflows
- `mc`（MinIO Client）

如果要做 CUDA 实战，还需要：

- 宿主机 `nvidia-smi` 正常
- Docker 支持 `--gpus all`

### 2. 写本次要生产什么

执行者主要维护 `batch plan`。

当前三应用 demo 已经写好：

```text
batch-plan.nvidia-cuda.practical-3apps.tsv
```

如果要生产别的应用，可以从模板复制一份：

```bash
cp batch-plan.template.tsv batch-plan.my-apps.tsv
```

然后只改里面的目标行。

### 3. 先预演

```bash
cd /home/xuefeng/newimage/images
make preview PLAN=你的plan.tsv
```

预演只会打印将执行的 `kubectl apply/create`，不会真的提交 Workflow。输出目录会自动变成 `dist/<plan文件名去掉.tsv>`。

### 4. 正式生产

```bash
make produce PLAN=你的plan.tsv
```

这条命令会自动完成：

```text
编译 matrix-ci
-> 运行 Go 测试
-> 打包 context.tar.gz
-> 上传 context 到 MinIO
-> 渲染 Argo WorkflowTemplate
-> 创建 Argo Workflow
```

### 5. 当前三应用 demo

如果只是跑当前仓库自带的三个 demo app：

```bash
make demo-preview
make demo
```

## batch plan 怎么写

当前 `plan` 一行格式是：

```text
hardware app_name app_version variant stages
```

注意：

- 这不是严格 TSV
- 代码里用的是空白分列，所以空格和 Tab 都能识别
- 最稳妥的写法就是“每列之间一个空格”

### 字段含义

1. `hardware`  
   比如：`nvidia`

2. `app_name`  
   必须和 `images-matrix.yaml` 里的应用名完全一致

3. `app_version`  
   必须和矩阵里的版本完全一致

4. `variant`  
   必须和矩阵里的 `variants.name` 完全一致

5. `stages`  
   可选。推荐只写用户意图：
   - `base`：生产并测试 base image
   - `app`：生产并测试 app image

如果不写第 5 列，默认就是 `all`。

### 推荐写法：只写 base 或 app，依赖自动展开

生产 app image 的标准写法是一行：

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
```

虽然这一行只写了 `app`，但程序会根据 `images-matrix.yaml` 里的 stage 依赖自动展开为：

```text
host_driver -> base_image -> base_test -> app_image -> app_test
```

也就是说，`host_driver` 会自动排在 base 和 app 前面；`base_image` 不会被忘掉；`base_test` 也会在 base 构建后执行；`app_test` 会在 app 构建后执行。

`push` 阶段已经删除，因为 Kaniko 在 `base_image` / `app_image` 构建完成时已经通过 `--destination` 把镜像推送到目标 Registry。额外保留一个 `push` 阶段只会增加误解。

如果只想单独生产并测试 base，可以写：

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 base
```

这一行会自动展开为：

```text
host_driver -> base_image -> base_test
```

这里仍然要写 `app_name/app_version/variant`，这是当前项目的设计债务：`base_image` 计划仍然借助 app 记录来定位精确 `base_ref`，而不是独立写 `base_ref`。这样做是为了避免组合爆炸。当前原则是：

```text
需要什么 app，就只生产它明确依赖的那个 base。
```

## application matrix 怎么维护

`application_matrix` 现在按“应用 -> 类型 -> 版本 -> 运行时 -> 驱动版本”的思路维护。

核心规则：

- 应用名只负责标识应用本体，例如 `comfyui`、`llama_factory`。
- `type` 提前写在应用顶层，例如 `service`、`cli`、`runtime`，后续 smoke 测试会按类型选择验收方式。
- 版本必须写具体版本号，不建议再写 `latest`，否则后续无法判断镜像到底对应哪一次上游状态。
- `runtime` 当前主要是 `cuda`，未来可以扩展 `cpu`、`cann` 等。
- 驱动版本必须写细，例如 `cuda-124`，不要只写 `cuda`。
- `base_ref` 必须指向精确 base，例如 `cuda124_base`，避免通过 app 名称或粗粒度 `cuda_base` 反推。

当前示例结构：

```yaml
application_matrix:
  practical_apps:
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

### 示例 1：只生产共享 base

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 base
```

说明：当前 `base` 计划仍然借用一条 app 记录来定位对应 base，但矩阵里已经把 `base_ref` 写成了精确的 `cuda124_base`，不再只写粗粒度的 `cuda_base`。`base_test` 会用构建出的 base image 做 CUDA runtime 冒烟测试。

### 示例 2：只生产一个 app

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
```

### 示例 3：一次生产多个 app

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
nvidia llama_factory 0.9.0 llama-factory-cli-cuda124 app
nvidia stable_diffusion 3.5 stable-diffusion-runtime-cuda124 app
```

## 常用命令

### 第一次应该怎么用

你第一次拿到项目时，优先记住这两条命令：

```bash
make preview PLAN=你的plan.tsv
make produce PLAN=你的plan.tsv
```

它们的区别非常重要：

| 命令 | 是否真实生产镜像 | 会做什么 | 适合什么时候用 |
| --- | --- | --- | --- |
| `make preview` | 不会 | 编译 `matrix-ci`，读取 plan，生成 Argo YAML，打印将要执行的 `kubectl` 命令 | 提交产线前检查计划是否正确 |
| `make produce` | 会 | 编译、测试、打包 context、上传 MinIO、创建 Argo Workflow、等待阶段结果 | 确认无误后正式生产 |

简单说：

```text
preview = 演练，不真的干活
produce = 正式提交产线，真的开始构建镜像
```

`OUT_DIR` 不需要手写。程序会自动把生成物放到：

```text
dist/<plan文件名去掉.tsv>
```

例如：

```text
PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
OUT_DIR=dist/batch-plan.nvidia-cuda.practical-3apps
```

### 查看 Makefile 帮助

```bash
make help
```

这个命令会打印当前默认的 `PLAN`、`MATRIX`、`OUT_DIR`、`UPLOAD_DEST`。

### 预演某个 plan

```bash
make preview PLAN=你的plan.tsv
```

`preview` 会做这些事：

1. 编译 `./bin/matrix-ci`
2. 读取 `PLAN`
3. 读取 `images-matrix.yaml`
4. 生成 WorkflowTemplate/Workflow YAML 到 `OUT_DIR`
5. 打印将要执行的 `kubectl apply/create` 命令

`preview` 不会做这些事：

1. 不打包 `context.tar.gz`
2. 不上传 MinIO
3. 不创建真实 Argo Workflow
4. 不启动 Kaniko Pod
5. 不生产镜像

### 按某个 plan 正式生产

```bash
make produce PLAN=你的plan.tsv
```

`produce` 会完整执行生产链：

```text
build -> test -> package -> upload -> create
```

对应含义是：

| 步骤 | Make target | 作用 |
| --- | --- | --- |
| 编译工具 | `build` | 编译 `./bin/matrix-ci` |
| 本地自测 | `test` | 执行 Go 单元测试，确认程序逻辑没坏 |
| 打包上下文 | `package` | 执行 `hack/package_context.sh`，生成 `dist/context/context.tar.gz` |
| 上传 MinIO | `upload` | 把 `context.tar.gz` 上传到 MinIO，供 Kaniko Pod 下载 |
| 创建 Workflow | `create` | 生成 YAML，提交 Argo Workflow，并等待 `host_driver/base_test/app_test` 结果 |

所以 `produce` 是正式生产命令。它会真的让 Argo/Kaniko 开始构建镜像。

### 预演当前三应用 demo

```bash
make demo-preview
```

等价于：

```bash
make preview PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
```

### 正式生产当前三应用 demo

```bash
make demo
```

等价于：

```bash
make produce PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
```

### 单独生产共享 base

```bash
make base-produce
```

等价于：

```bash
make produce PLAN=batch-plan.nvidia-cuda.base12.4.tsv
```

### 只做底层调试时才需要的命令

```bash
make build
make test
make package
make upload
```

这些命令一般不是普通使用者入口：

| 命令 | 作用 | 是否常用 |
| --- | --- | --- |
| `make build` | 只编译 `matrix-ci` | 调试 Go 代码时用 |
| `make test` | 只跑 Go 单元测试 | 改代码后用 |
| `make package` | 只打包 context | 检查构建上下文时用 |
| `make upload` | 打包并上传 context 到 MinIO | 检查 MinIO/context 链路时用 |
| `make render` | 只生成 YAML，不提交 | 检查 Argo YAML 时用 |
| `make apply` | 只提交 WorkflowTemplate，不创建 Workflow | Argo 模板调试时用 |
| `make create` | 创建 Workflow 并等待结果，但不自动打包/上传 context | 已确认 context 最新时用 |

## matrix-ci 和 Makefile 的角色

使用者优先使用 Makefile：

```bash
make produce PLAN=你的plan.tsv
```

`matrix-ci batch` 是 Makefile 内部调用的底层命令：

```bash
./bin/matrix-ci batch \
  --plan ./batch-plan.nvidia-cuda.practical-3apps.tsv \
  --matrix ./images-matrix.yaml \
  --out-dir ./dist/batch-plan.nvidia-cuda.practical-3apps \
  --create
```

`matrix-ci batch` 实际做的事情是：

1. 读取 `plan`
2. 读取 `matrix`
3. 把目标解析成内部构建单元
4. 生成 Argo WorkflowTemplate
5. 根据参数执行 `kubectl apply/create`
6. 默认等待 Workflow 结束，并输出 `host_driver/base_test/app_test` 等关键阶段的反馈

如果 `host_driver` 失败，后续不会继续；如果 `base_test` 失败，说明 base image 不达标，后续 app 生产也会被取消；如果 `app_test` 失败，说明镜像已经构建出来但应用级冒烟没有通过。三者都会在命令行摘要中直接显示。

Makefile 额外负责固定前置步骤：

```text
build -> test -> package -> upload -> matrix-ci batch --create
```

## Smoke 测试

当前 `hack/test/smoke.sh` 保留三层：

### L0

固定检查基础环境：

- `python3`
- `pip`

### L1

当前 L1 采用“四分类 + 轻量验证”：

- `runtime`
  - `import + version/关键类`
- `cli`
  - `--help` 或 `--version`
- `service`
  - `入口文件存在` 或 `关键模块 import`
- `workspace`
  - `主命令 version/help` 或 `关键模块 import`

当前仓库已经实例化的 3 个样本：

- `comfyui` -> `service`
  - `entrypoint` 存在
  - `import folder_paths`
- `llama_factory` -> `cli`
  - `llamafactory-cli version` 或 `--help`
- `stable_diffusion` -> `runtime`
  - `diffusers.__version__`
  - `StableDiffusion3Pipeline`

完整分类见：

- `docs/application-taxonomy.md`

### L2

当前 L2 只做到 NVIDIA CUDA runtime 级别，目标是覆盖所有 CUDA 镜像，而不是验证某个具体框架或模型。

- 容器内 `nvidia-smi` 可用
- 容器内能看到 GPU
- 容器内能读取 NVIDIA driver version
- 容器内能看到 `libcuda.so.1`
- 容器内能看到 CUDA runtime 相关动态库，例如 `libcudart.so`

注意：

- L1 只做轻量冒烟
- L2-runtime 不依赖 `torch`，也不会下载模型或跑推理
- 真实 `docker run` 是否能直接落地，不算在当前 L2-runtime 里

`host_driver` 和 L2-runtime 的区别：

- `host_driver` 检查节点是否有资格跑目标 CUDA 镜像，例如驱动版本是否满足 `cuda-124`。
- L2-runtime 检查构建出来的 app image 在容器内部是否真的能看到 CUDA 驱动链路。

## 当前三个应用的实际状态

### comfyui

- 已经属于可直接落地的服务镜像
- 可直接 `docker run`
- 已实测可返回 `HTTP 200`

### llama_factory

- 产线构建、推送、L1 smoke 已通过
- 但默认 WebUI 直跑仍可能受依赖版本错位影响
- 问题主要在应用依赖组合，不在 Kaniko 产线

### stable_diffusion

- 当前是 `diffusers` 运行时镜像
- 可 import、可用关键 pipeline
- 但默认 `docker run` 不会自己起服务
- 它现在不是成品服务镜像

## 当前边界

这套仓库当前明确不做这些事：

- 不自动生成所有 Dockerfile
- 不试图用矩阵描述全部应用安装细节
- 不把所有异构硬件同时拉进一条主线

它当前最适合做的事是：

**在已经有 Dockerfile 的前提下，稳定地把 host_driver/base/app/app_test 这条链跑通。**

## 已知设计债务

### 1. base plan 仍然借助 app 记录反推 base

这在语义上不够干净，但当前可用。

### 2. stable_diffusion 仍然只是 runtime image

如果后续要把它变成可交付服务镜像，需要先确定它的服务形态。

### 3. llama_factory 仍需要进一步 pin 依赖

构建成功不等于默认启动一定成功。

## 推荐工作方式

后续扩 app 时，建议固定按这个顺序：

1. 查官方仓库 / 官方 Dockerfile / 官方安装文档
2. 判断它属于：
   - `runtime`
   - `cli`
   - `service`
   - `workspace`
3. 写 Dockerfile 和 entrypoint
4. 在 `images-matrix.yaml` 中登记
5. 写一条最小 `plan`
6. 先执行 `make preview PLAN=你的plan.tsv`
7. 再执行 `make produce PLAN=你的plan.tsv`
8. 最后再做 `docker run` 落地验证

## 参考文档

- `docs/practical-app-dockerfile-sources.md`
- `docs/application-taxonomy.md`
- `docs/host-driver-check-research.md`
- `hack/local-registry/README.md`
