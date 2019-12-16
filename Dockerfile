FROM golang:1.12-alpine AS development

ENV PROJECT_PATH=/wazigate-lora
ENV PATH=$PATH:$PROJECT_PATH/build
ENV CGO_ENABLED=1

WORKDIR /
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    make \
    git \
    bash \
    gcc \
    libc-dev \
    linux-headers \
    && cd / \
    && git clone https://github.com/Lora-net/lora_gateway.git \
    && cd lora_gateway/libloragw/ \
    && make libloragw.a \
    && mkdir -p $PROJECT_PATH

COPY . $PROJECT_PATH
WORKDIR $PROJECT_PATH
RUN mv /lora_gateway/libloragw/libloragw.a SX1301/libs \
    && export branch=$(git rev-parse --abbrev-ref HEAD); \
    export version=$(git describe --always); \
    go build -ldflags "-s -w -X main.version=$version -X main.branch=$branch" -o build/wazigate-lora .

#--------------#

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates tzdata curl
COPY --from=development /wazigate-lora/build/wazigate-lora .
COPY www www/
ENTRYPOINT ["./wazigate-lora", "-r", "sx127x"]