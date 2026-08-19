FROM golang AS golang

WORKDIR /root/

# Install dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Build the binary
COPY internal internal
COPY cmd cmd
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags "-s -w" -buildvcs=false -o wazigate-lora ./cmd/wazigate-lora

#

FROM scratch

COPY --from=golang /root/wazigate-lora /wazigate-lora

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD [ "/wazigate-lora", "healthcheck" ]

ENTRYPOINT ["/wazigate-lora"]
