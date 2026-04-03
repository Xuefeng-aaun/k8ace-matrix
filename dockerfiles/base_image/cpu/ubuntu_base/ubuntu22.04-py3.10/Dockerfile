ARG BASE_IMAGE=ubuntu:22.04
FROM ${BASE_IMAGE}

ARG APT_MIRROR=https://mirrors.tuna.tsinghua.edu.cn
ARG PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple
ARG PIP_EXTRA_INDEX_URL=

ENV DEBIAN_FRONTEND=noninteractive

RUN if [ -f /etc/apt/sources.list ]; then \
      sed -i "s|http://archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g; s|http://security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g; s|https://archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g; s|https://security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
    fi && \
    apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl git python3 python3-pip python3-venv && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc && \
    printf '[global]\nindex-url = %s\n' "${PIP_INDEX_URL}" > /etc/pip.conf && \
    if [ -n "${PIP_EXTRA_INDEX_URL}" ]; then printf 'extra-index-url = %s\n' "${PIP_EXTRA_INDEX_URL}" >> /etc/pip.conf; fi

COPY hack/lib /opt/k8ace/hack/lib

