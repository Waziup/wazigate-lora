FROM golang:1.26.6 AS golang

WORKDIR /root/

# Install dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Build the binary
COPY internal internal
COPY cmd cmd
ENV CGO_ENABLED=0
# Go's signal-based async preemption (SIGURG) can livelock under qemu-user: the
# build pins one core at 100% indefinitely instead of compiling. Costs nothing on
# a healthy toolchain and removes the failure mode under emulation.
ENV GODEBUG=asyncpreemptoff=1
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags "-s -w" -buildvcs=false -o wazigate-lora ./cmd/wazigate-lora

FROM scratch

COPY --from=golang /root/wazigate-lora /wazigate-lora

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD [ "/wazigate-lora", "healthcheck" ]

ENTRYPOINT ["/wazigate-lora"]
