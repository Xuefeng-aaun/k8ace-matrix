ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG http_proxy
ARG https_proxy
ARG no_proxy

ENV PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1 \
    HF_HOME=/opt/huggingface \
    HF_HUB_DISABLE_TELEMETRY=1

RUN python3 -m pip install --upgrade pip && \
    python3 -m pip install \
      --index-url https://download.pytorch.org/whl/cu124 \
      torch==2.5.1 \
      torchvision==0.20.1 \
      torchaudio==2.5.1 && \
    python3 -m pip install \
      --index-url https://pypi.org/simple \
      "diffusers[torch]==0.37.0" \
      transformers \
      accelerate \
      safetensors \
      sentencepiece \
      invisible-watermark

RUN mkdir -p /opt/k8ace/hack/test /workspace/stable-diffusion
COPY hack/test/smoke.sh /opt/k8ace/hack/test/smoke.sh
RUN chmod +x /opt/k8ace/hack/test/smoke.sh

WORKDIR /workspace/stable-diffusion
CMD ["python3"]
