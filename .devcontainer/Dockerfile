FROM golang:1.26.4-bookworm

COPY scripts/install-apt-deps.sh /tmp/install-apt-deps.sh
RUN chmod +x /tmp/install-apt-deps.sh && /tmp/install-apt-deps.sh && rm /tmp/install-apt-deps.sh

ENV PATH="/go/bin:${PATH}"
ENV GOTOOLCHAIN=auto

WORKDIR /workspace