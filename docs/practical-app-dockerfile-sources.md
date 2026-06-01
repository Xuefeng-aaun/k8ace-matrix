# Practical App Dockerfile Sources

本仓库当前保留的三个实战应用，都采用“参考官方来源，再按本地产线改写”的方式维护 Dockerfile。

## comfyui

- 官方仓库：<https://github.com/comfy-org/ComfyUI>
- 当前仓库内选择的安装方式：
  - `git clone --branch v0.22.0`
  - 使用官方 `requirements.txt`
- 本地化改写：
  - 不使用上游现成基础镜像，统一改为 `ARG BASE_IMAGE`
  - 固定接入 `cuda12.4` 共享 base
  - 复制 `hack/test/smoke.sh`
  - 复用本地 `hack/images/comfyui/entrypoint.sh`

## llama_factory

- 官方仓库：<https://github.com/hiyouga/LLaMA-Factory>
- 参考依据：
  - 官方仓库源码结构
  - 官方 `docker/docker-cuda/Dockerfile` 的思路
- 当前仓库内选择的安装方式：
  - `git clone --branch v0.9.0`
  - 安装官方 `requirements.txt`
  - 执行 `pip install -e ".[metrics]"`
- 本地化改写：
  - 统一改为 `ARG BASE_IMAGE`
  - 固定使用 `cuda12.4` 共享 base
  - 增加本地 `entrypoint.sh`
  - 保留 `llamafactory-cli` 作为应用级 smoke 验收入口

## stable_diffusion

- 参考依据：<https://huggingface.co/docs/diffusers/installation>
- 当前仓库内选择的安装方式：
  - 安装 `diffusers[torch]==0.37.0`
  - 补齐 `transformers`、`accelerate`、`safetensors`、`sentencepiece`、`invisible-watermark`
- 本地化改写：
  - 当前保留的是一个可复用的 Stable Diffusion 运行时镜像，不内嵌特定 WebUI
  - 统一改为 `ARG BASE_IMAGE`
  - 固定接入 `cuda12.4` 共享 base
  - 以 `StableDiffusion3Pipeline` 作为当前应用级 smoke 验证入口
