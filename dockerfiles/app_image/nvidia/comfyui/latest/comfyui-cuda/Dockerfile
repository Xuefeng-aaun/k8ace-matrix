ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG COMFYUI_REPO=https://github.com/comfyanonymous/ComfyUI.git
ARG COMFYUI_REF=master
ARG TORCH_INDEX_URL=https://download.pytorch.org/whl/cu124

ENV COMFYUI_ROOT=/opt/comfyui

RUN apt-get update && \
    apt-get install -y --no-install-recommends git && \
    rm -rf /var/lib/apt/lists/*

RUN git clone "${COMFYUI_REPO}" "${COMFYUI_ROOT}" && \
    cd "${COMFYUI_ROOT}" && \
    git checkout "${COMFYUI_REF}"

RUN python3 -m venv "${COMFYUI_ROOT}/venv" && \
    "${COMFYUI_ROOT}/venv/bin/python" -m pip install -U pip && \
    "${COMFYUI_ROOT}/venv/bin/pip" install --index-url "${TORCH_INDEX_URL}" torch torchvision torchaudio && \
    if [ -f "${COMFYUI_ROOT}/requirements.txt" ]; then "${COMFYUI_ROOT}/venv/bin/pip" install -r "${COMFYUI_ROOT}/requirements.txt"; fi

ENV PATH=${COMFYUI_ROOT}/venv/bin:${PATH}

COPY hack/images/comfyui /opt/k8ace/hack/images/comfyui
RUN chmod +x /opt/k8ace/hack/images/comfyui/entrypoint.sh

EXPOSE 8188

ENTRYPOINT ["/opt/k8ace/hack/images/comfyui/entrypoint.sh"]

