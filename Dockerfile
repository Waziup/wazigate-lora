FROM golang:1.13-alpine AS development

ENV PROJECT_PATH=/wazigate-lora
ENV PATH=$PATH:$PROJECT_PATH/build
ENV CGO_ENABLED=0
ENV GO_EXTRA_BUILD_ARGS="-a -installsuffix cgo"

RUN apk add --no-cache ca-certificates make git bash alpine-sdk

RUN mkdir -p $PROJECT_PATH
COPY . $PROJECT_PATH
COPY .git/HEAD $PROJECT_PATH
WORKDIR $PROJECT_PATH

RUN export branch=$(git rev-parse --abbrev-ref HEAD);
RUN export version=$(git describe --always);

RUN go build -ldflags "-s -w -X main.version=$version -X main.branch=$branch" -o build/wazigate-lora .

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates curl
COPY --from=development /wazigate-lora/build/wazigate-lora .
COPY www www/
ENTRYPOINT ["./wazigate-lora"]
