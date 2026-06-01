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
  base_image/nvidia/cuda_base/
  app_image/nvidia/comfyui/
  app_image/nvidia/llama_factory/
  app_image/nvidia/stable_diffusion/

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
- Docker
- kubectl
- Argo Workflows
- `mc`（MinIO Client）

如果要做 CUDA 实战，还需要：

- 宿主机 `nvidia-smi` 正常
- Docker 支持 `--gpus all`

### 2. 先编译并跑测试

```bash
cd /home/xuefeng/newimage/images
make build
make test
```

### 3. 先生产共享 base

```bash
make base-create
```

### 4. base 成功后，再生产 app

```bash
make practical-create
```

### 5. 第一次自己写 plan 时，先 dry-run

```bash
./bin/matrix-ci batch \
  --plan ./你的plan.tsv \
  --matrix ./images-matrix.yaml \
  --out-dir ./dist/argo-check \
  --create --dry-run
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
   可选。用逗号分隔。  
   例如：
   - `base_image`
   - `app_image,test,push`

如果不写第 5 列，默认就是 `all`。

### 示例 1：只生产共享 base

```text
# hardware app_name app_version variant stages
nvidia comfyui latest comfyui-cuda base_image
```

说明：当前 `base` 计划仍然借用一条 app 记录来反推对应 base，这是现有实现下的一个设计债务。

### 示例 2：只生产一个 app

```text
# hardware app_name app_version variant stages
nvidia comfyui latest comfyui-cuda app_image,test,push
```

### 示例 3：一次生产多个 app

```text
# hardware app_name app_version variant stages
nvidia comfyui latest comfyui-cuda app_image,test,push
nvidia llama_factory 0.9.0 llamafactory-cuda app_image,test,push
nvidia stable_diffusion 3.5 sd-cuda app_image,test,push
```

## 常用命令

### 编译

```bash
make build
```

### 测试

```bash
make test
```

### 打包 context

```bash
make package
```

### 上传 context

```bash
make upload
```

### 渲染 base workflow

```bash
make base-render
```

### 创建 base workflow

```bash
make base-create
```

### 渲染三个实战 app

```bash
make practical-render
```

### 创建三个实战 app

```bash
make practical-create
```

### 只做 dry-run

```bash
make dry-run
```

## matrix-ci 的角色

当前最重要的命令是：

```bash
./bin/matrix-ci batch \
  --plan ./batch-plan.nvidia-cuda.practical-3apps.tsv \
  --matrix ./images-matrix.yaml \
  --out-dir ./dist/argo-practical-3apps \
  --create
```

它实际做的事情是：

1. 读取 `plan`
2. 读取 `matrix`
3. 把目标解析成内部构建单元
4. 生成 Argo WorkflowTemplate
5. 如有需要，先打包并上传 context
6. 最后创建真正 Workflow

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

当前只做 NVIDIA 基础可见性检查：

- `nvidia-smi`
- `torch.cuda.is_available()`

注意：

- L1 只做轻量冒烟
- 真实 `docker run` 是否能直接落地，不算在当前 L1 里

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

**在已经有 Dockerfile 的前提下，稳定地把 base/app/test/push 这条链跑通。**

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
6. 先 dry-run
7. 再跑真实 workflow
8. 最后再做 `docker run` 落地验证

## 参考文档

- `docs/practical-app-dockerfile-sources.md`
- `docs/application-taxonomy.md`
- `hack/local-registry/README.md`
