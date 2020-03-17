FROM golang:1.13-alpine AS development

ENV CGO_ENABLED=0

RUN apk add --no-cache ca-certificates git \
    && mkdir -p /wazigate-lora

COPY . /wazigate-lora
# COPY .git/HEAD $PROJECT_PATH
WORKDIR /wazigate-lora

# RUN export branch=$(git rev-parse --abbrev-ref HEAD);
# RUN export version=$(git describe --always);

RUN go build -a -installsuffix cgo -ldflags "-s -w" -o build/wazigate-lora .

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates curl
COPY --from=development /wazigate-lora/build/wazigate-lora .
COPY forwarders ./forwarders/
ENTRYPOINT ["./wazigate-lora"]
