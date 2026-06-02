# Host Driver 检查模块前置调研

本文用于指导后续 `host_driver` 阶段从占位阶段升级为真实检查阶段。

当前结论：三应用 demo 已先实现 NVIDIA CUDA 的 `host_driver` 检查。CANN、CNML、ROCm/DTK、MXMACA 暂不进入代码，只保留调研结论，避免在缺少真机输出和官方映射表时做出错误阻断。

调研日期：2026-06-03

## 目标边界

`host_driver` 要检查的是“目标节点是否具备运行某类加速镜像的宿主环境”，不是检查 Dockerfile 内容，也不是检查 Kaniko 能不能构建镜像。

建议把检查分成四层：

| 层级 | 名称 | 目的 | 失败后是否应阻断 |
|---|---|---|---|
| H0 | 工具存在 | 检查管理工具是否可执行，例如 `nvidia-smi`、`npu-smi`、`mx-smi` | 是 |
| H1 | 设备存在 | 检查机器是否识别到目标加速卡 | 是 |
| H2 | 版本读取 | 读取驱动、固件、运行时或 SDK 版本 | 是 |
| H3 | 兼容性判断 | 判断目标 base/app 要求的运行时版本是否被当前驱动支持 | 视生态而定 |

对于当前三应用 demo，`host_driver` 会检查 NVIDIA GPU 是否可见、`nvidia-smi` 是否可用、宿主驱动版本是否足以支撑目标 CUDA 版本。这个检查必须有用：如果目标是 `cuda-124`，但宿主驱动只满足 `cuda-118`，则应提前失败，因为后续 smoke 即使构建完成也无法有效验收。

## 总体设计建议

不同硬件不要共用一套固定命令，而应采用“生态类型 -> 检查策略”的方式。当前代码只启用 NVIDIA 策略。

建议未来增加类似配置：

```yaml
host_driver_matrix:
  nvidia:
    runtime: "cuda"
    command: "nvidia-smi"
    device_check:
      args: ["-L"]
    version_check:
      args:
        - "--query-gpu=driver_version,name,compute_cap"
        - "--format=csv,noheader"
    compatibility:
      type: "min_driver"
      rules:
        cuda-118:
          min_driver_linux: "520.61.05"
        cuda-124:
          min_driver_linux: "550.54.15"
        cuda-131:
          min_driver_linux: "590.44.01"

  ascend:
    runtime: "cann"
    command: "npu-smi"
    device_check:
      args: ["info"]
    compatibility:
      type: "vendor_mapping"
      mapping_source: "CANN Version <-> Ascend HDK Version"
```

注意：这只是设计草案，不建议现在直接写死到 `images-matrix.yaml`。原因是 `images-matrix.yaml` 当前主要承担镜像生产资产，不应过早混入大量节点探测细节。

## NVIDIA CUDA

### 推荐检查项

| 检查 | 推荐命令 | 说明 |
|---|---|---|
| 工具存在 | `command -v nvidia-smi` | `nvidia-smi` 随 NVIDIA 驱动提供 |
| 设备存在 | `nvidia-smi -L` | 能列出 GPU 即通过基础检查 |
| 驱动版本 | `nvidia-smi --query-gpu=driver_version,name,compute_cap --format=csv,noheader` | 可解析驱动版本、GPU 名称、算力架构 |
| CUDA 兼容 | 根据目标 `cuda-xxx` 查最小 driver | 这是 NVIDIA 最适合自动化判断的部分 |

### 兼容性规则

NVIDIA 是这几个生态里最适合做自动化兼容判断的。核心原因是它明确说明 CUDA Driver 具有向后兼容性：应用基于某个 CUDA 版本编译后，通常可以在后续更新的驱动上运行。

当前可以先维护这些规则：

| 目标 CUDA | 最小 Linux Driver | 备注 |
|---|---:|---|
| CUDA 11.8 | `520.61.05` | CUDA 11.8 GA 对应最小驱动 |
| CUDA 12.4 GA | `550.54.14` | CUDA 12.4 GA |
| CUDA 12.4 Update 1 | `550.54.15` | 当前项目可采用这个更保守的值 |
| CUDA 13.1 GA | `590.44.01` | 如后续新增 `cuda131_base` 可用 |

NVIDIA 的检查逻辑可以写成：

```text
如果 command 不存在 -> FAIL
如果 nvidia-smi -L 无设备 -> FAIL
读取 driver_version
根据 base_ref/runtime 选择最低驱动版本
如果 driver_version >= min_driver -> PASS
否则 FAIL
```

### 资料依据

- NVIDIA Container Wiki 说明：运行 CUDA 容器只要求宿主机有兼容驱动，宿主机不必安装 CUDA Toolkit。
- NVIDIA CUDA Release Notes 说明：每个 CUDA Toolkit 需要最低 CUDA Driver，且 CUDA Driver 对后续驱动具备向后兼容性。

## 华为 Ascend CANN

### 推荐检查项

| 检查 | 推荐命令/路径 | 说明 |
|---|---|---|
| 工具存在 | `command -v npu-smi` | Ascend 管理工具 |
| 设备存在 | `npu-smi info` | 检查 NPU 是否可见、Health 是否正常 |
| 驱动版本 | `/usr/local/Ascend/driver/version.info` | 常见驱动版本文件 |
| 固件/板卡 | `npu-smi info -t board` | 可读取板卡、固件等信息 |
| CANN 环境 | `/usr/local/Ascend/ascend-toolkit/set_env.sh` | 容器或宿主环境经常需要 source |

### 兼容性规则

CANN 不建议照搬 NVIDIA 的“新驱动兼容旧运行时”思路。更稳妥的判断方式是维护官方 CANN Version 与 Ascend HDK Version 的映射表。

例如华为云 ModelArts Lite Server 文档中列出了 CANN 与 Ascend HDK 的组件兼容关系，如 CANN 8.0.0 对应 Ascend HDK 24.1.0、24.1.RC 系列、23.0.X 等；CANN 8.2.RC1 对应 Ascend HDK 25.2.0、25.0.RC1、24.1.0 等。

建议检查逻辑：

```text
如果 npu-smi 不存在 -> FAIL
如果 npu-smi info 无 NPU 或 Health 异常 -> FAIL
读取 driver/HDK 版本
读取目标 CANN 版本
查 vendor_mapping 是否允许该组合
命中映射 -> PASS
未命中 -> WARN 或 FAIL，由策略决定
```

### 容器注意事项

CANN 容器通常不仅需要设备文件，还需要挂载部分驱动库和工具。常见对象包括：

- `/dev/davinci*`
- `/dev/davinci_manager`
- `/dev/devmm_svm`
- `/dev/hisi_hdc`
- `/usr/local/Ascend/driver`
- `npu-smi`

所以 CANN 的 `host_driver` 最好不只检查宿主，还要在未来增加“容器内是否能看到驱动库”的检查。

## 寒武纪 CNML / Neuware

### 推荐检查项

| 检查 | 推荐方向 | 说明 |
|---|---|---|
| 工具存在 | CNMon | 寒武纪官方文档中心列出 CNMon 用户手册 |
| SDK/运行时 | CNToolkit、CNRT、CNML | 官方文档中心列出 CNToolkit、CNRT、CNML 手册 |
| 设备存在 | CNMon 设备列表 | 具体命令需要以企业版手册或真机输出为准 |
| 运行时可用 | CNRT 初始化/设备数量 | CNRT 是寒武纪运行时库 |

### 兼容性规则

寒武纪公开网页能确认 CNMon、CNToolkit、CNML、CNRT 等组件存在，但具体版本兼容表和命令输出格式公开资料较少，且部分 SDK 下载面向企业用户。

因此不要现在硬编码 CNML 检查命令。建议先设计为“待真机校准”的策略：

```text
如果 CNMon 工具不存在 -> FAIL
如果设备列表为空 -> FAIL
读取 CNToolkit/CNML/CNRT 版本 -> PASS/WARN
兼容性判断暂不自动 fail，先记录版本供人工确认
```

等拿到寒武纪机器或企业手册后，再把输出格式写成解析器。

## 海光 DCU / ROCm / DTK

### 推荐检查项

| 检查 | 推荐方向 | 说明 |
|---|---|---|
| 工具存在 | `hy-smi`、`rocminfo`、`rocm-smi` 或 `amd-smi` | 具体取决于海光 DTK/ROCm 发行环境 |
| 驱动存在 | amdgpu/hygon 相关内核驱动 | ROCm 风格生态通常依赖内核驱动加载 |
| 设备存在 | SMI 工具设备列表 | 需要真机确认命令输出 |
| ROCm/DTK 版本 | `/opt/rocm` 或 DTK 路径 | 需要和框架镜像版本匹配 |

### 兼容性规则

海光 DCU 通常被描述为 ROCm/DTK 路线，但它不是标准 AMD ROCm 环境的简单复制。AMD 官方 AMD SMI 文档可以作为“ROCm 风格检查”的参考：AMD SMI 初始化要求 `amdgpu` 驱动已加载，并提供 CLI/API 用于监控 GPU。

但海光自身的 `hy-smi`、DTK 版本、驱动版本和框架镜像之间的精确关系，需要海光官方文档或真机确认。当前建议：

```text
不要把 AMD ROCm 的兼容矩阵直接套给海光
先把海光策略标记为 rocm_like
检查工具存在和设备存在
版本兼容先只记录，不自动阻断
拿到海光官方 DTK/driver mapping 后再启用强校验
```

## 沐曦 MXMACA

### 推荐检查项

| 检查 | 推荐命令/路径 | 说明 |
|---|---|---|
| PCI 设备 | `lspci | grep 9999` | OpenCloudOS 适配文档中使用此方式检查 GPU 设备识别 |
| 工具存在 | `mx-smi` 或 `/opt/mxdriver/bin/mx-smi` | OpenCloudOS 文档说明 mx-smi 在驱动包路径下 |
| 版本读取 | `mx-smi --show-version` | 可读取当前固件/驱动相关版本 |
| 驱动包 | `yum list installed | grep metax-driver` | 检查已安装驱动包 |
| SDK | `maca_sdk` | MXMACA SDK 安装包 |

### 兼容性规则

MXMACA 更适合采用 vendor mapping，而不是 NVIDIA 式的简单最小版本判断。原因是沐曦生态涉及驱动、MXMACA SDK、框架适配包、模型框架版本等多层组合。

建议检查逻辑：

```text
如果 mx-smi 不存在 -> FAIL
如果 lspci 未识别沐曦设备 -> FAIL
读取 mx-smi --show-version
读取 metax-driver / maca_sdk 版本
如果存在明确 vendor mapping -> 判断 PASS/FAIL
否则只做 WARN，提示需要人工确认镜像与驱动版本是否匹配
```

## 当前实现状态

当前不再是纯 noop。

NVIDIA CUDA 已实现：

- `command -v nvidia-smi`
- `nvidia-smi -L`
- `nvidia-smi --query-gpu=driver_version --format=csv,noheader`
- 根据 `accelerator` 或 base 元数据判断最低 Linux Driver。
- Pod 会申请 `nvidia.com/gpu: "1"`，确保检查结果来自真实可分配 GPU。

CANN、MXMACA、寒武纪、海光暂不进入代码，只保留调研文档。

保留统一 `host_driver` 阶段仍然有价值，因为未来可以按 `hardware/runtime` 分发到不同厂商策略。

## 后续实现路线

### 第一阶段：NVIDIA CUDA

先实现 NVIDIA，因为资料最清晰，兼容规则最成熟。

最低实现：

```bash
command -v nvidia-smi
nvidia-smi -L
nvidia-smi --query-gpu=driver_version,name,compute_cap --format=csv,noheader
```

再根据 `base_ref` 或 `accelerator` 判断最低驱动版本。

### 第二阶段：CANN / MXMACA

实现 vendor mapping，但先不强制覆盖全部版本。

例如：

```yaml
compatibility:
  type: vendor_mapping
  allow:
    cann-800:
      hdk:
        - "24.1.0"
        - "24.1.RC3"
        - "24.1.RC2"
```

### 第三阶段：寒武纪 / 海光

先通过真实机器采集输出：

```text
工具路径
工具 version 输出
设备列表输出
驱动版本输出
SDK/Toolkit 版本输出
容器内挂载要求
```

拿到输出样本后再写解析器，不提前硬编码。

## 资料来源

- NVIDIA CUDA Compatibility: https://docs.nvidia.com/deploy/cuda-compatibility/
- NVIDIA CUDA Toolkit Release Notes: https://docs.nvidia.com/cuda/cuda-toolkit-release-notes/
- NVIDIA Container Wiki: https://nvidia.github.io/container-wiki/toolkit/container-images.html
- Huawei Cloud ModelArts Lite Server User Guide: https://support.huaweicloud.com/intl/en-us/usermanual-server-modelarts/
- 寒武纪开发者社区文档中心: https://developer.cambricon.com/index/document/index/classid/3.html
- AMD SMI Documentation: https://rocmdocs.amd.com/projects/amdsmi/en/latest/
- 沐曦 MXMACA 驱动本地部署参考文档: https://developer.metax-tech.com/doc/157
- OpenCloudOS 沐曦环境部署文档: https://docs.opencloudos.org/en/OC9/ai-deployment/GPU-optimization-practice/metax-deployment/
