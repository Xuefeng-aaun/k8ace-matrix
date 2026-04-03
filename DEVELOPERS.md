# 开发者指南：新增镜像与 CI 流程

本指南说明如何在本仓库新增不同层次的镜像（基础镜像/业务镜像/阶段镜像）、如何编写镜像内嵌脚本（kubernetes/hack 风格），以及如何让 CI（Argo Workflows + Kaniko）“自动发现并构建”新增内容。

## 一、目录与规约（必须遵守）

### 1) Matrix（唯一事实来源）

- Matrix 配置文件： [images-matrix.yaml](file:///Users/jarvis/Documents/trae_projects/images/images-matrix.yaml)
- 建议把“要构建什么”放在：
  - `base_image_matrix`
  - `application_matrix`
  - `build_pipeline.stages`
  - `build_args_overrides`
- 加速卡硬件矩阵（不同厂商字段差异大且变化频率低）已迁移到 Go 常量：
  - `internal/hardware`（示例入口：AllMatrices）
- 硬件矩阵维护原则：
  - 每个“架构（Architecture）”一个 Go 文件：`internal/hardware/<vendor>_<arch>.go`
  - 不把任何硬件细节写回 `images-matrix.yaml`
  - 推荐驱动统一用 `DriverVersion.Extensions["recommended"]=true`
  - 非标字段统一写入 `Extensions map[string]any`（避免为少数厂商字段污染统一结构）
- 把“如何用 Argo Workflows 跑”放在：
  - `ci_cd.argo_workflows`（namespace、service_account、kaniko_image、registry_secret_name、cache repo 模板等）

### 1.1) 硬件矩阵（internal/hardware）如何维护

硬件矩阵用于表达：
- 卡架构与支持的卡型号（例如 NVIDIA Ampere: A100_40GB/A100_80GB）
- 推荐驱动与支持驱动列表/区间（不同厂商命名差异大）
- 运行时与 Kubernetes 相关的非标信息（device plugin、vgpu、工具链版本等）

核心类型位于：
- [types.go](file:///Users/jarvis/Documents/trae_projects/images/internal/hardware/types.go)

聚合入口位于：
- [builtin.go](file:///Users/jarvis/Documents/trae_projects/images/internal/hardware/builtin.go)

字段约定：
- 驱动“列表”用 `ArchitectureInfo.DriverVersions` 表达（适用于版本离散、需要逐个标注 recommended/eol/branch 的场景）
- 驱动“区间”用 `ArchitectureInfo.DriverRange` 表达（适用于版本连续、只关心范围的场景；具体版本细节可放 Extensions）
- 推荐驱动：
  - 统一在对应的 `DriverVersion` 上设置 `Extensions["recommended"]=true`
  - `GetBestDriverVersion()` 会优先选择被标记为 recommended 的版本
- 非标字段：
  - 一律写入 `Extensions`（示例：`cuda_versions`/`rocm_versions`/`oneapi`/`dtk`/`branch`/`eol`/`notes`）

新增一个架构的最小步骤：
- 1) 在 `internal/hardware/` 新增文件：`<vendor>_<arch>.go`
- 2) 在该文件中定义一个仅包含该架构的 `HardwareMatrix` 常量
- 3) 在 [builtin.go](file:///Users/jarvis/Documents/trae_projects/images/internal/hardware/builtin.go) 中把该常量合并进对应 vendor 的矩阵（mergeMatrices）
- 4) 跑 `go test ./...` 与 `go vet ./...`，并用 `matrix-ci render` 验证关键路径

架构文件模板（示意）：

```go
package hardware

var ExampleVendorExampleArch = HardwareMatrix{
	Vendor: VendorNVIDIA,
	Architectures: map[Architecture][]DeviceModel{
		ArchAmpere: {
			{
				Name:         "A100_80GB",
				Architecture: ArchAmpere,
				DriverMatrix: []DriverVersion{
					{
						Version: "550.54.14",
						Extensions: map[string]any{
							"recommended": true,
							"cuda_versions": []string{
								"12.4",
							},
						},
					},
				},
			},
		},
	},
	ArchitectureInfo: map[Architecture]ArchitectureInfo{
		ArchAmpere: {
			Description:    "NVIDIA Ampere",
			SupportedCards: []string{"A100_40GB", "A100_80GB"},
			DriverVersions: []string{"535.161.07", "550.54.14"},
		},
	},
}
```

### 2) Dockerfile 目录（与 Go/CI 强绑定）

matrix-ci 渲染出来的 Workflow 会引用下面的 Dockerfile 路径规则：

- **基础镜像（base_image stage）**
  - `dockerfiles/base_image/<hardware>/<base_ref>/<tag_suffix>/Dockerfile`
  - 示例：  
    - [cuda base Dockerfile](file:///Users/jarvis/Documents/trae_projects/images/dockerfiles/base_image/nvidia/cuda_base/cuda12.4-devel-ubuntu22.04/Dockerfile)

- **业务镜像（app_image stage）**
  - `dockerfiles/app_image/<hardware>/<app>/<version>/<variant>/Dockerfile`
  - 示例：  
    - [sd_webui Dockerfile](file:///Users/jarvis/Documents/trae_projects/images/dockerfiles/app_image/nvidia/sd_webui/1.10.0/sd-webui-cuda/Dockerfile)
    - [comfyui Dockerfile](file:///Users/jarvis/Documents/trae_projects/images/dockerfiles/app_image/nvidia/comfyui/latest/comfyui-cuda/Dockerfile)

如果你新增镜像但不按该路径放 Dockerfile，CI 会找不到文件。

### 3) 镜像内嵌脚本（kubernetes/hack 风格）

该仓库采用 kubernetes/hack 的模块化脚本风格，约定如下：

- 统一入口： [hack/lib/init.sh](file:///Users/jarvis/Documents/trae_projects/images/hack/lib/init.sh)
  - 负责设置 bash 严格选项并加载依赖库
- 库脚本：
  - [hack/lib/logging.sh](file:///Users/jarvis/Documents/trae_projects/images/hack/lib/logging.sh)（`k8ace::log::*`）
  - [hack/lib/util.sh](file:///Users/jarvis/Documents/trae_projects/images/hack/lib/util.sh)（`k8ace::util::*`）
  - [hack/lib/gpu.sh](file:///Users/jarvis/Documents/trae_projects/images/hack/lib/gpu.sh)（`k8ace::gpu::*`）
- 业务镜像入口脚本放在：
  - `hack/images/<app>/entrypoint.sh`
  - 示例：
    - [sd-webui entrypoint](file:///Users/jarvis/Documents/trae_projects/images/hack/images/sd-webui/entrypoint.sh)
    - [comfyui entrypoint](file:///Users/jarvis/Documents/trae_projects/images/hack/images/comfyui/entrypoint.sh)

脚本风格硬性要求：
- 第一行：`#!/usr/bin/env bash`
- 开启：`errexit/nounset/pipefail`
- 函数命名空间：`k8ace::<domain>::<verb>`
- 不输出敏感信息（不要打印 token/password/.dockerconfigjson 内容）

## 二、新增基础镜像（base_image）

### 目标

让 matrix-ci 能推导出 base 组合，并在 Argo Workflow 里通过 Kaniko 构建基础镜像。

### 步骤

1) 在 `images-matrix.yaml` 新增或更新 `base_image_matrix.<base_ref>.variants[]`
- `base_ref` 是逻辑名称（例如 `cuda_base`、`rocm_base`、`ubuntu_base`、`oneapi_base`）
- `tag_suffix` 决定 Dockerfile 目录名与最终镜像 tag 片段
- `k8ace_compatible` 用于匹配硬件（例如 `["nvidia"]` / `["hygon"]` / `["intel"]`）

2) 新增对应 Dockerfile（必须按路径放）
- `dockerfiles/base_image/<hardware>/<base_ref>/<tag_suffix>/Dockerfile`

3) 让 base 镜像可“被覆盖”
- Dockerfile 顶部统一使用：
  - `ARG BASE_IMAGE=...`
  - `FROM ${BASE_IMAGE}`
- CI 侧通过 build args 覆盖 `BASE_IMAGE`，避免把镜像源写死。

4) 中国区镜像源（默认清华源）
- 建议基础镜像支持以下 build args：
  - `APT_MIRROR`（默认清华）
  - `PIP_INDEX_URL`（默认清华）
  - `PIP_EXTRA_INDEX_URL`（可选）

## 三、新增业务镜像（app_image）

### 目标

让 matrix-ci 能从 `application_matrix` 推导出 (app, version, variant) 组合，并在 Argo Workflow 里构建业务镜像（默认依赖上游 base_image 产物）。

### 步骤

1) 在 `images-matrix.yaml` 增加 `application_matrix` 条目
- 建议结构：
  - `application_matrix.<category>.<app>.versions[]`
  - `application_matrix.<category>.<app>.variants[]`
    - `name`：variant 名称（用于目录与输出）
    - `base_ref`：引用 base_image_matrix 的 key
    - `hardware`：支持的硬件列表
    - `additional_packages`：用于生成/参考安装清单
    - `build_args`：支持 `${version}` 与 `${base.<field>}` 占位符

2) 新增 Dockerfile
- `dockerfiles/app_image/<hardware>/<app>/<version>/<variant>/Dockerfile`

3) 新增 entrypoint（推荐）
- `hack/images/<app>/entrypoint.sh`
- Dockerfile 内：
  - `COPY hack/images/<app> /opt/k8ace/hack/images/<app>`
  - `ENTRYPOINT ["/opt/k8ace/hack/images/<app>/entrypoint.sh"]`

4) 版本策略
- 如果版本是 `latest`，目录也使用 `latest/`，保持一致。

## 四、新增 CI 阶段（build_pipeline.stages）

### 现状与约定

目前 pipeline 至少包含 `base_image` 与 `app_image`。其他阶段（test/push 等）在本仓库默认以“占位”方式存在（任务会生成，但 Dockerfile 可能是 noop）。

### 新增阶段的最小流程

1) 在 `images-matrix.yaml` 增加 stage（并声明 depends_on）
2) 在 `dockerfiles/<stage>/noop/Dockerfile` 提供占位（如果你暂时没有真实实现）
3) 如果要实现真实逻辑，新增对应 stage 的 Dockerfile 与镜像内脚本（建议放 hack/lib 或 hack/images/<app> 下）

## 五、CI（Argo Workflows + Kaniko）落地流程

### 1) registry 凭证（Kaniko push 必需）

- 在 `images-matrix.yaml` 配置：
  - `ci_cd.argo_workflows.registry_secret_name: <secret>`
- Secret 类型应为 `kubernetes.io/dockerconfigjson`，并包含 key：`.dockerconfigjson`
- 渲染出的 Workflow 会把它映射为 `/kaniko/.docker/config.json`

### 2) 渲染 WorkflowTemplate 到目录（推荐）

```bash
go run ./cmd/matrix-ci render \
  --matrix ./images-matrix.yaml \
  --hardware nvidia \
  --app-name sd_webui \
  --app-version 1.10.0 \
  --variant sd-webui-cuda \
  --out-dir dist/argo
```

输出默认会按 build unit 拆分为多个文件（便于 GitOps 管理）：
- `dist/argo/workflowtemplates/<name>.yaml`

便捷脚本（sd_webui 示例）：
```bash
bash hack/render_sd.sh
```

### 3) 提交执行（可选）

```bash
export MATRIX_CI_ARGO_TOKEN="***"
go run ./cmd/matrix-ci submit \
  --matrix ./images-matrix.yaml \
  --namespace default \
  --argo-server https://argo.example.com \
  --hardware nvidia \
  --app-name sd_webui \
  --app-version 1.10.0 \
  --variant sd-webui-cuda
```

注意：submit 目前要求选择条件最终只命中一个 build unit（避免一次提交多个 workflow）。

## 六、开发自检清单

- Dockerfile 路径是否符合规约（否则 CI 会找不到）
- base/app 的 `ARG BASE_IMAGE` 是否存在并正确使用
- apt/pip 镜像源是否可通过参数覆盖
- entrypoint 是否 `source hack/lib/init.sh` 并使用命名空间函数
- registry secret 是否仅引用名称、不输出内容
- 本地执行：
  - `go test ./...`
  - `go vet ./...`
