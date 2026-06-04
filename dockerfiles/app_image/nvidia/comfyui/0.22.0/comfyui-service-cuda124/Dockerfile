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
    COMFYUI_ROOT=/opt/comfyui \
    PYTHONPATH=/opt/comfyui \
    K8ACE_HACK_ROOT=/opt/k8ace/hack \
    PORT=8188 \
    LISTEN=0.0.0.0 \
    K8ACE_SMOKE_TYPE=service \
    K8ACE_SMOKE_SERVICE_CMD=/opt/k8ace/hack/images/comfyui/entrypoint.sh \
    K8ACE_SMOKE_PORT=8188 \
    K8ACE_SMOKE_HEALTH_PATH=/ \
    K8ACE_SMOKE_TIMEOUT_SECONDS=180

RUN python3 -m pip install --upgrade pip && \
    python3 -m pip install \
      --index-url https://download.pytorch.org/whl/cu124 \
      torch==2.5.1 \
      torchvision==0.20.1 \
      torchaudio==2.5.1 && \
    git clone --depth 1 --branch v0.22.0 https://github.com/comfy-org/ComfyUI.git ${COMFYUI_ROOT} && \
    python3 -m pip install \
      --index-url https://pypi.org/simple \
      -r ${COMFYUI_ROOT}/requirements.txt && \
    python3 -m pip check

RUN mkdir -p /opt/k8ace/hack/test /opt/k8ace/hack/lib /opt/k8ace/hack/images/comfyui
COPY hack/test/smoke.sh /opt/k8ace/hack/test/smoke.sh
COPY hack/lib/ /opt/k8ace/hack/lib/
COPY hack/images/comfyui/entrypoint.sh /opt/k8ace/hack/images/comfyui/entrypoint.sh
RUN chmod +x /opt/k8ace/hack/test/smoke.sh /opt/k8ace/hack/images/comfyui/entrypoint.sh

WORKDIR /opt/comfyui
EXPOSE 8188
ENTRYPOINT ["/opt/k8ace/hack/images/comfyui/entrypoint.sh"]
