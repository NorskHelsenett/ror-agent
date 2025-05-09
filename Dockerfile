ARG GCR_MIRROR=gcr.io/
FROM ${GCR_MIRROR}distroless/static:nonroot
LABEL org.opencontainers.image.source https://github.com/norskhelsenett/ror-agent
LABEL org.opencontainers.image.description ROR Agent v1
WORKDIR /

COPY cmd/agent/version.json /version.json
COPY dist/ror-cluster-agent /bin/ror-agent
USER 10000:10000
EXPOSE 8100

ENTRYPOINT ["/bin/ror-agent"]

