FROM registry.access.redhat.com/ubi9/go-toolset:latest as builder
ENV GOPATH=$APP_ROOT
COPY --chown=1001:0 . .
RUN make cmd

FROM registry.access.redhat.com/ubi9/ubi-minimal
USER root
WORKDIR /working
COPY --from=builder /opt/app-root/src/bin/addon /usr/local/bin/addon
ENTRYPOINT ["/usr/local/bin/addon"]
