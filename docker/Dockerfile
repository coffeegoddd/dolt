# syntax=docker/dockerfile:1.3-labs
FROM ubuntu:22.04

ARG DOLT_VERSION

RUN apt update -y && \
    apt install -y \
        curl \
        tini \
        ca-certificates && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*

RUN bash -c 'curl -L https://github.com/dolthub/dolt/releases/download/v${DOLT_VERSION}/install.sh | bash'
RUN /usr/local/bin/dolt version

WORKDIR /var/lib/dolt
ENTRYPOINT ["tini", "--", "/usr/local/bin/dolt"]

