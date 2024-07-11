# syntax=docker/dockerfile:1.3-labs
FROM ubuntu:22.04

ARG DOLT_VERSION=1.41.3

RUN apt update -y && \
    apt install -y \
        curl \
        tini \
        ca-certificates && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*

# RUN bash -c 'curl -L https://github.com/dolthub/dolt/releases/v${DOLT_VERSION}/download/install.sh | bash'
RUN curl -LO curl -L https://github.com/dolthub/dolt/releases/v${DOLT_VERSION}/download/install.sh
RUN bash -c ./install.sh
RUN /usr/local/bin/dolt version

WORKDIR /var/lib/dolt
ENTRYPOINT ["tini", "--", "/usr/local/bin/dolt"]

