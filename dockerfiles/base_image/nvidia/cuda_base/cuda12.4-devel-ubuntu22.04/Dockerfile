ARG BASE_IMAGE=nvidia/cuda:12.4-devel-ubuntu22.04
FROM ${BASE_IMAGE}

ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG NO_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG no_proxy=
ARG APT_MIRROR=
ARG PIP_INDEX_URL=https://pypi.org/simple
ARG PIP_EXTRA_INDEX_URL=

ENV HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY} \
    http_proxy=${http_proxy} \
    https_proxy=${https_proxy} \
    no_proxy=${no_proxy}

ENV DEBIAN_FRONTEND=noninteractive

RUN if [ -n "${APT_MIRROR}" ] && [ -f /etc/apt/sources.list ]; then \
      sed -i "s|http://archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g; s|http://security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g; s|https://archive.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g; s|https://security.ubuntu.com/ubuntu/|${APT_MIRROR}/ubuntu/|g" /etc/apt/sources.list; \
    fi && \
    apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl git python3 python3-pip python3-venv && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc && \
    printf '[global]\nindex-url = %s\n' "${PIP_INDEX_URL}" > /etc/pip.conf && \
    if [ -n "${PIP_EXTRA_INDEX_URL}" ]; then printf 'extra-index-url = %s\n' "${PIP_EXTRA_INDEX_URL}" >> /etc/pip.conf; fi

COPY hack/lib /opt/k8ace/hack/lib

