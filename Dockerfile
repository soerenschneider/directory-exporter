
FROM golang:1.20.0 as builder
WORKDIR /build/
ADD . /build/
RUN go build -ldflags="-X main.BuildVersion=$(git describe --tags --abbrev=0 || echo dev) -X main.CommitHash=$(git rev-parse HEAD)" -o directory-exporter main.go

FROM gcr.io/distroless/base
COPY --from=builder "/build/directory-exporter" /directory-exporter
USER 65532:65532
ENTRYPOINT ["/directory-exporter"]
