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
    LLAMAFACTORY_ROOT=/opt/LLaMA-Factory \
    GRADIO_SERVER_NAME=0.0.0.0 \
    GRADIO_SERVER_PORT=7860 \
    API_PORT=8000 \
    K8ACE_SMOKE_TYPE=service \
    K8ACE_SMOKE_SERVICE_CMD=/opt/k8ace/hack/images/llama-factory/entrypoint.sh \
    K8ACE_SMOKE_PORT=7860 \
    K8ACE_SMOKE_HEALTH_PATH=/ \
    K8ACE_SMOKE_TIMEOUT_SECONDS=180

RUN printf '%s\n' \
      'gradio==4.44.1' \
      'gradio-client==1.3.0' \
      'pydantic<3.0.0' \
      'fastapi<1.0.0' \
      'starlette<1.0.0' \
      > /tmp/llamafactory-constraints.txt && \
    python3 -m pip install --upgrade pip setuptools wheel && \
    python3 -m pip install \
      --index-url https://download.pytorch.org/whl/cu124 \
      torch==2.5.1 \
      torchvision==0.20.1 \
      torchaudio==2.5.1 && \
    git clone --depth 1 --branch v0.9.0 https://github.com/hiyouga/LLaMA-Factory.git ${LLAMAFACTORY_ROOT} && \
    python3 -m pip install \
      --index-url https://pypi.org/simple \
      --constraint /tmp/llamafactory-constraints.txt \
      -r ${LLAMAFACTORY_ROOT}/requirements.txt && \
    cd ${LLAMAFACTORY_ROOT} && \
    python3 -m pip install \
      --index-url https://pypi.org/simple \
      --constraint /tmp/llamafactory-constraints.txt \
      -e ".[metrics]" && \
    python3 -c 'import gradio; print("gradio", gradio.__version__)' && \
    python3 -m pip check

RUN mkdir -p /opt/k8ace/hack/test /opt/k8ace/hack/images/llama-factory
COPY hack/test/smoke.sh /opt/k8ace/hack/test/smoke.sh
COPY hack/images/llama-factory/entrypoint.sh /opt/k8ace/hack/images/llama-factory/entrypoint.sh
RUN chmod +x /opt/k8ace/hack/test/smoke.sh /opt/k8ace/hack/images/llama-factory/entrypoint.sh

WORKDIR /opt/LLaMA-Factory
EXPOSE 7860 8000
ENTRYPOINT ["/opt/k8ace/hack/images/llama-factory/entrypoint.sh"]
