ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG NO_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG no_proxy=
ARG SD_WEBUI_REPO=https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
ARG SD_WEBUI_REF=master
ARG TORCH_INDEX_URL=https://download.pytorch.org/whl/cu124

ENV HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY} \
    http_proxy=${http_proxy} \
    https_proxy=${https_proxy} \
    no_proxy=${no_proxy}

ENV SD_WEBUI_ROOT=/opt/sd-webui

RUN apt-get update && \
    apt-get install -y --no-install-recommends git && \
    rm -rf /var/lib/apt/lists/*

RUN git clone "${SD_WEBUI_REPO}" "${SD_WEBUI_ROOT}" && \
    cd "${SD_WEBUI_ROOT}" && \
    git checkout "${SD_WEBUI_REF}"

RUN python3 -m venv "${SD_WEBUI_ROOT}/venv" && \
    "${SD_WEBUI_ROOT}/venv/bin/python" -m pip install -U pip && \
    "${SD_WEBUI_ROOT}/venv/bin/pip" install --index-url "${TORCH_INDEX_URL}" torch torchvision torchaudio && \
    if [ -f "${SD_WEBUI_ROOT}/requirements_versions.txt" ]; then "${SD_WEBUI_ROOT}/venv/bin/pip" install -r "${SD_WEBUI_ROOT}/requirements_versions.txt"; fi

ENV PATH=${SD_WEBUI_ROOT}/venv/bin:${PATH}

COPY hack/images/sd-webui /opt/k8ace/hack/images/sd-webui
RUN chmod +x /opt/k8ace/hack/images/sd-webui/entrypoint.sh

EXPOSE 7860

ENTRYPOINT ["/opt/k8ace/hack/images/sd-webui/entrypoint.sh"]

