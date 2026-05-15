FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        curl \
        dbus-x11 \
        git \
        libfuse2 \
        libgtk-3-dev \
        libwebkit2gtk-4.1-dev \
        nsis \
        pkg-config \
        rpm \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*
