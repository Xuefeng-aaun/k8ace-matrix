ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG http_proxy
ARG https_proxy
ARG no_proxy

ENV DEBIAN_FRONTEND=noninteractive \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1

RUN if [ -f /etc/apt/sources.list ]; then \
      sed -i 's|http://security.ubuntu.com/ubuntu/|http://archive.ubuntu.com/ubuntu/|g; s|https://archive.ubuntu.com/ubuntu/|http://archive.ubuntu.com/ubuntu/|g; s|https://security.ubuntu.com/ubuntu/|http://archive.ubuntu.com/ubuntu/|g' /etc/apt/sources.list; \
    fi && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      python3 \
      python3-pip \
      python3-venv && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc && \
    printf '[global]\nindex-url = https://pypi.org/simple\n' > /etc/pip.conf
