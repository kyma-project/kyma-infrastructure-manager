FROM golang:1.23.0-alpine3.20 as build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workdir

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/


ARG BIN
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o runtime-cleanup cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /workdir/runtime-cleanup .
USER 65532:65532

ENTRYPOINT ["/runtime-cleanup"]
