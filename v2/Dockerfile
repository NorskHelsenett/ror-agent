ARG GCR_MIRROR=gcr.io/
FROM ${GCR_MIRROR}distroless/static:nonroot
LABEL org.opencontainers.image.source="https://github.com/norskhelsenett/ror-agent"
LABEL org.opencontainers.image.description="ROR Agent v2"
WORKDIR /

COPY dist/ror-cluster-agent /bin/ror-cluster-agent
USER 10000:10000
EXPOSE 8100

ENTRYPOINT ["/bin/ror-cluster-agent"]

