FROM scratch

LABEL org.opencontainers.image.source="https://github.com/supernetes/supernetes"
LABEL org.opencontainers.image.description="Kubernetes bridge for Supercomputers"
LABEL org.opencontainers.image.licenses="MPL-2.0"

USER 1000

COPY ../bin/controller /
ENTRYPOINT ["/controller"]
