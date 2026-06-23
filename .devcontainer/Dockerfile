FROM golang:1.26.4-bookworm

COPY scripts/install-apt-deps.sh /tmp/install-apt-deps.sh
RUN chmod +x /tmp/install-apt-deps.sh \
  && /tmp/install-apt-deps.sh \
  && rm /tmp/install-apt-deps.sh

ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=${USER_UID}

RUN groupadd --gid ${USER_GID} ${USERNAME} \
  && useradd --uid ${USER_UID} --gid ${USER_GID} -m -s /bin/bash ${USERNAME}

ENV GOPATH=/home/${USERNAME}/go \
  GOCACHE=/home/${USERNAME}/.cache/go-build \
  GOMODCACHE=/home/${USERNAME}/go/pkg/mod \
  PATH=/home/${USERNAME}/go/bin:/usr/local/go/bin:${PATH} \
  GOTOOLCHAIN=auto

RUN mkdir -p \
    ${GOPATH}/bin \
    ${GOPATH}/pkg/mod \
    /home/${USERNAME}/.cache/go-build \
    /workspace \
  && chown -R ${USERNAME}:${USERNAME} /home/${USERNAME} /workspace

WORKDIR /workspace
USER ${USERNAME}