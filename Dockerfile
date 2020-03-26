FROM golang:1.13-alpine AS development

ENV CGO_ENABLED=0

COPY . /wazigate-lora
WORKDIR /wazigate-lora

RUN apk add --no-cache ca-certificates git zip \
    && mkdir -p /wazigate-lora \
    && zip -r index.zip app \
    && go build -a -installsuffix cgo -ldflags "-s -w" -o build/wazigate-lora .

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates curl

COPY --from=development /wazigate-lora/build/wazigate-lora .
COPY --from=development /wazigate-lora/index.zip /index.zip

COPY www www
COPY app/conf/wazigate-lora /etc/wazigate-lora

ENTRYPOINT ["./wazigate-lora"]
