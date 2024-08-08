FROM golang:latest
ENV GIN_MODE=release

ARG TARGETPLATFORM
COPY artifacts/build/release/$TARGETPLATFORM/* /bin/

ENTRYPOINT ["/bin/browser"]
