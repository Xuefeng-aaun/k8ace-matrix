# 2026-06-03 本地设计回顾

本文记录从下面这句对话开始，本地工作期间对 `k8ace-matrix` 项目做出的设计整理：

> 把工作路径暂时改到 E:\docker\final\k8ace-matrix-main，我暂时连不上服务器了，先在本地进行工作，然后把对矩阵文件的更改应用到本地

这轮工作的核心不是单纯修 bug，而是重新梳理项目的使用方式和产线语义，让这个项目从“能跑但难懂”变成“使用者知道自己要写什么、程序知道该怎么展开执行”。

## 一、当时的主要问题

在切到本地工作前，项目已经能在服务器上通过 Argo、Kaniko、MinIO、本地 Registry 跑出镜像，但存在几个明显问题。

第一，`images-matrix.yaml` 的 application 层语义不够精确。原先很多 app 的 `base_ref` 只写到 `cuda_base` 这种粗粒度类型，无法明确说明应用到底依赖 `cuda-118`、`cuda-121`、`cuda-124` 还是未来的 `cuda-131`。这会导致后续维护时继续依赖“从 app 反推 base”的模糊逻辑。

第二，版本命名不适合长期维护。部分配置容易使用 `latest` 这类不可追踪版本，这会让后续无法判断某个镜像到底对应哪一次上游状态，也不利于审查和复现。

第三，用户入口太复杂。原先使用者需要理解并手写 `host_driver`、`base_image`、`base_test`、`app_image`、`test`、`push` 等多个 stage，导致 batch plan 看起来像是在写底层 DAG，而不是在表达“我要生产什么”。

第四，`push` 阶段语义多余。Kaniko 在执行 `base_image` 或 `app_image` 构建时，已经通过 `--destination` 把镜像推送到 Registry。额外保留 `push` stage 容易让人误以为镜像是在最后一个独立阶段才入库。

第五，命令行反馈不足。Argo Workflow 创建后，如果 `host_driver`、`base_test` 或应用冒烟测试失败，使用者很难从命令行立即知道到底是哪一关失败。

第六，Makefile 不够像使用者入口。`preview`、`produce` 的区别不清楚，还要求手动写 `OUT_DIR`，对新手不友好。

## 二、这一轮确立的新原则

这轮本地修改确立了几个新的设计原则。

第一，矩阵文件是人工维护资产，不再试图承担“一键自动生成全部 Dockerfile”的幻想。Dockerfile 可以由人工或 AI 参考官方资料编写，矩阵负责登记当前仓库承诺支持的目标。

第二，batch plan 只表达使用者意图，不直接表达底层 DAG。使用者只需要写 `base` 或 `app`，程序负责把它展开为完整链路。

第三，产线必须有前置门禁。`host_driver` 不是可有可无的占位步骤，它应该在 `base_image` 和 `app_image` 前面，用于确认当前节点的 NVIDIA GPU、驱动和目标 CUDA 版本兼容。

第四，base image 也必须有冒烟测试。只有 `base_test` 通过，才说明这个 base image 具备后续 app image 构建与测试的基础。若 base 不达标，后续 app 生产没有意义。

第五，应用测试阶段应该命名为 `app_test`，而不是笼统的 `test`。这样它和 `base_test` 对称，Argo 节点、日志、文档都更清楚。

第六，删除独立 `push` 阶段。镜像是否入库由 Kaniko 的构建阶段完成，不再额外保留一个容易误导的空阶段。

## 三、application matrix 的新结构

这一轮将 `application_matrix` 按下面的层级重新组织：

```text
app_name
  -> type
  -> versions
     -> app_version
        -> runtimes
           -> runtime
              -> accelerator_version
```

这个结构用于回答五个问题：

1. 这个应用是什么类型，例如 `service`、`cli`、`runtime`、`workspace`。
2. 这个应用的具体版本是什么，不再使用 `latest`。
3. 这个应用跑在哪类运行时上，例如当前主要是 `cuda`。
4. 这个应用绑定哪条 CUDA 版本，例如 `cuda-124`。
5. 这个应用依赖哪个精确 base，例如 `cuda124_base`。

当前三个 demo app 被整理为：

```text
comfyui
  type: service
  version: 0.22.0
  variant: comfyui-service-cuda124
  base_ref: cuda124_base

llama_factory
  type: cli
  version: 0.9.0
  variant: llama-factory-cli-cuda124
  base_ref: cuda124_base

stable_diffusion
  type: runtime
  version: 3.5
  variant: stable-diffusion-runtime-cuda124
  base_ref: cuda124_base
```

这相当于把矩阵文件从“模糊分类表”变成了“人工审核过的生产清单”。

## 四、base 与 app 的依赖关系

这一轮明确了当前项目仍然存在一个设计债务：`base` 计划仍然需要借助 `app_name/app_version/variant` 来定位对应 base。

也就是说，单独生产 base 时仍然要写类似：

```text
nvidia comfyui 0.22.0 comfyui-service-cuda124 base
```

从语义上看，这不是最完美的形式。更理想的写法可能是直接声明：

```text
nvidia cuda124_base base
```

但当前项目还没有实现完全独立的 base plan 语法，所以暂时保留“通过 app variant 找到 base_ref”的方式。

这样做的好处是能避免组合爆炸。当前原则是：

```text
需要什么 app，就生产它明确依赖的那个 base。
```

这比盲目全量生成所有 base 和所有 app 的组合更保守，也更适合当前项目阶段。

## 五、构建链路的新语义

本轮修改后，底层构建链路固定为：

```text
host_driver -> base_image -> base_test -> app_image -> app_test
```

其中各阶段含义如下：

| 阶段 | 作用 |
| --- | --- |
| `host_driver` | 检查 NVIDIA GPU 是否可见、`nvidia-smi` 是否可用、驱动版本是否满足目标 CUDA 最低要求 |
| `base_image` | 使用 Kaniko 构建共享基础镜像 |
| `base_test` | 启动刚构建的 base image，检查 Python、pip、GPU 可见性、CUDA runtime 动态库 |
| `app_image` | 基于已构建的 base image 构建应用镜像 |
| `app_test` | 启动刚构建的 app image，执行 L1 应用冒烟和 L2 CUDA runtime 冒烟 |

用户层不再直接手写这一长串。

如果使用者写：

```text
nvidia comfyui 0.22.0 comfyui-service-cuda124 base
```

程序自动展开为：

```text
host_driver -> base_image -> base_test
```

如果使用者写：

```text
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
```

程序自动展开为：

```text
host_driver -> base_image -> base_test -> app_image -> app_test
```

这就是当前 batch plan 的新使用模型。

## 六、host_driver 的定位

`host_driver` 被重新确认为真正的前置门禁，而不是为了保持完整链路而存在的空 stage。

当前只支持 NVIDIA CUDA，因此 `host_driver` 的检查策略也只实现 NVIDIA：

1. 请求 `nvidia.com/gpu: "1"`。
2. 使用目标 CUDA base image 作为检查容器。
3. 检查 `nvidia-smi` 是否存在。
4. 检查 Pod 内是否能看到 GPU。
5. 读取节点 NVIDIA driver version。
6. 根据目标 CUDA 版本判断驱动是否满足最低要求。

例如 `cuda-124` 对应的最低 Linux 驱动版本为：

```text
550.54.15
```

如果 `host_driver` 失败，说明当前节点没有资格继续生产这个目标镜像，后续 base 和 app 都不应该继续。

## 七、base_test 的意义

这轮讨论中明确了一点：base image 不应该只构建成功就算合格。

`base_test` 会启动刚构建出来的 base image，并检查：

1. `python3 --version`
2. `pip --version` 或 `pip3 --version`
3. `nvidia-smi`
4. GPU 是否可见
5. driver version 是否能读取
6. `libcuda.so.1` 是否可见
7. `libcudart.so` 或相关 CUDA runtime 库是否可见

如果 `base_test` 失败，说明 base image 本身不具备后续 app 的运行基础。这时即使继续构建 app，也大概率只是制造一个不可用产物。

因此 `app_image` 现在依赖 `base_test`，而不是直接依赖 `base_image`。

## 八、app_test 的定位

`app_test` 取代了原先笼统的 `test` 阶段。

当前 `app_test` 调用：

```bash
/opt/k8ace/hack/test/smoke.sh L2 <app_name> nvidia
```

它包含三层含义：

1. L0：检查基础命令，例如 `python3`、`pip`。
2. L1：按应用类型做轻量应用级冒烟。
3. L2：检查 NVIDIA CUDA runtime 是否在容器内可见。

当前三个 demo 的 L1 逻辑是：

| 应用 | 类型 | L1 冒烟 |
| --- | --- | --- |
| `comfyui` | `service` | 检查 entrypoint 和关键模块 |
| `llama_factory` | `cli` | 检查 `llamafactory-cli version/help` |
| `stable_diffusion` | `runtime` | 检查 `diffusers` 和关键 pipeline import |

L2 暂时不跑真实模型推理，因为真实推理会引入模型下载、显存、启动参数等额外变量，不适合作为当前最小统一冒烟测试。

## 九、删除 push 阶段

删除 `push` 阶段是这轮设计里很重要的一步。

原因是 Kaniko 构建阶段已经包含：

```text
--destination=<目标镜像地址>
```

所以 `base_image` 和 `app_image` 只要构建成功，就已经把镜像推送到目标 Registry。

继续保留 `push` 阶段会产生三个问题：

1. 让使用者误以为镜像是在最后的 `push` 阶段才提交。
2. 让 DAG 看起来比实际复杂。
3. 增加一个没有真实职责的阶段，后续排查时容易混淆。

因此新的主链不再有 `push`。

## 十、batch plan 的新写法

当前推荐的 batch plan 非常简单。

生产一个 base：

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 base
```

生产一个 app：

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
```

生产三个 demo app：

```text
# hardware app_name app_version variant stages
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
nvidia llama_factory 0.9.0 llama-factory-cli-cuda124 app
nvidia stable_diffusion 3.5 stable-diffusion-runtime-cuda124 app
```

这背后的设计目标是：执行者只关心“我要生产什么”，不关心底层 DAG 怎么拼。

## 十一、命令行反馈机制

这轮还增强了 `matrix-ci batch --create` 的反馈逻辑。

现在它会：

1. 先根据 plan 生成 WorkflowTemplate 和 Workflow。
2. 执行 `kubectl apply`。
3. 执行 `kubectl create`。
4. 先把本批次所有 Workflow 都提交出去。
5. 再等待每个 Workflow 的最终结果。
6. 输出关键阶段摘要。

之所以先全部提交再等待，是为了保留 Argo 并发运行能力。如果每创建一个 Workflow 就立刻等待，会把多条产线串行化，不符合“同时看到多条产线在跑”的目标。

反馈中会重点展示：

```text
host_driver
base_image
base_test
app_image
app_test
```

如果 `host_driver`、`base_test` 或 `app_test` 失败，命令会返回错误，而不是沉默结束。

## 十二、Makefile 的使用者入口

本轮最后一个重要变化是重新整理 Makefile。

原先使用者需要写：

```bash
make preview PLAN=你的plan.tsv OUT_DIR=dist/argo-check
make produce PLAN=你的plan.tsv OUT_DIR=dist/argo-check
```

后来进一步简化为：

```bash
make preview PLAN=你的plan.tsv
make produce PLAN=你的plan.tsv
```

`OUT_DIR` 不再需要手写，会自动变成：

```text
dist/<plan文件名去掉.tsv>
```

例如：

```text
PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv
OUT_DIR=dist/batch-plan.nvidia-cuda.practical-3apps
```

这让使用者只需要关注 `PLAN`，不用再额外维护输出目录命名。

## 十三、preview 和 produce 的区别

这轮文档里专门解释了 `preview` 和 `produce`。

`preview` 是演练：

1. 编译 `matrix-ci`。
2. 读取 plan。
3. 读取 matrix。
4. 生成 Argo YAML。
5. 打印将要执行的 `kubectl apply/create` 命令。

`preview` 不会：

1. 打包 context。
2. 上传 MinIO。
3. 创建真实 Workflow。
4. 启动 Kaniko Pod。
5. 生产镜像。

`produce` 是正式生产：

```text
build -> test -> package -> upload -> create
```

含义是：

| 步骤 | 作用 |
| --- | --- |
| `build` | 编译 `matrix-ci` |
| `test` | 跑 Go 单元测试 |
| `package` | 打包 `context.tar.gz` |
| `upload` | 上传 context 到 MinIO |
| `create` | 创建 Argo Workflow 并等待结果 |

因此后续真实使用时，推荐流程是：

```bash
make preview PLAN=你的plan.tsv
make produce PLAN=你的plan.tsv
```

先演练，再生产。

## 十四、当前项目的重新定位

这一轮之后，项目定位进一步收敛为：

```text
Dockerfile 由人工或 AI 编写
images-matrix.yaml 负责登记支持范围
batch plan 负责声明本次生产目标
matrix-ci 负责展开 stage、生成 Argo YAML、提交 Workflow、反馈结果
Argo/Kaniko/MinIO/Registry 负责真实构建和产物落库
```

项目不再追求：

1. 由矩阵自动生成所有 Dockerfile。
2. 全量生产所有 app 和所有 base 的组合。
3. 把所有异构硬件同时纳入当前主线。

当前只聚焦：

```text
NVIDIA CUDA 生态下，已有 Dockerfile 的 base/app 镜像批量生产与最小冒烟验证。
```

这个定位比原先更保守，但更清楚、更可执行。

## 十五、已经完成的验证

本地完成了以下验证：

```bash
go test ./... -count=1
```

通过。

本地也通过等价命令验证了自动输出目录：

```bash
./bin/matrix-ci.exe batch \
  --plan batch-plan.nvidia-cuda.practical-3apps.tsv \
  --matrix images-matrix.yaml \
  --out-dir dist/batch-plan.nvidia-cuda.practical-3apps \
  --create \
  --dry-run
```

输出结果确认三条 demo app 会生成对应 WorkflowTemplate 和 Workflow。

由于本地没有 `make` 命令，所以没有直接执行 Makefile，但已经验证了 Makefile 对应的底层 `matrix-ci batch` 行为。

## 十六、当前仍然存在的设计债务

第一，base plan 仍然借助 app 记录反推 base。这是当前最明显的语义债务。未来如果要进一步优化，可以增加独立 base plan 语法。

第二，`matrix-ci batch --create` 的等待反馈依赖 `kubectl get workflow -o json`。这要求目标环境已经安装 Argo CRD，并且 kubectl 能访问对应 namespace。

第三，当前 `host_driver` 只支持 NVIDIA。虽然之前讨论过 CANN、ROCm、MXMACA 等异构生态，但当前版本没有展开。

第四，L2-runtime 仍然只是 CUDA runtime 检查，不代表真实业务推理一定成功。它适合作为统一最小冒烟，但不是最终落地验收。

第五，Dockerfile 质量仍然是项目核心风险。现在产线能把 Dockerfile 变成镜像，但 Dockerfile 本身是否真正好用，仍然需要按应用逐个设计和验证。

## 十七、今晚设计的结论

这几个小时的核心成果可以概括为一句话：

```text
把项目从“使用者手写复杂 stage 链路”调整为“使用者只写 base/app 意图，程序自动展开、执行并反馈结果”。
```

对应的最终使用方式是：

```bash
make preview PLAN=你的plan.tsv
make produce PLAN=你的plan.tsv
```

对应的 batch plan 写法是：

```text
nvidia comfyui 0.22.0 comfyui-service-cuda124 app
```

对应的底层真实链路是：

```text
host_driver -> base_image -> base_test -> app_image -> app_test
```

这轮改动让项目更适合作为一个面向使用者的半自动镜像产线，而不是一个只有开发者才能理解的脚本集合。
