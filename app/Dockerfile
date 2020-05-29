FROM golang:1.13-alpine AS development

ENV CGO_ENABLED=0

COPY . /wazigate-lora
WORKDIR /wazigate-lora

RUN apk add --no-cache ca-certificates git zip \
    && cd app \
    && zip -q -r ../index.zip . \
    && cd /wazigate-lora \
    && go build -a -installsuffix cgo -ldflags "-s -w" -o wazigate-lora .

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates curl

COPY --from=development /wazigate-lora/wazigate-lora .
COPY --from=development /wazigate-lora/index.zip /index.zip

COPY ../www/dist www/dist
COPY ../www/img www/img
COPY conf/wazigate-lora /etc/wazigate-lora

ENTRYPOINT ["./wazigate-lora"]
