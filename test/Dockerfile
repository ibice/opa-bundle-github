FROM docker.io/golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN \
  --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  CGO_ENABLED=0 go build ./cmd/opa-bundle-github

FROM alpine
COPY --from=builder /build/opa-bundle-github /usr/local/bin/opa-bundle-github
ENTRYPOINT [ "/usr/local/bin/opa-bundle-github" ]