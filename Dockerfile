FROM golang:1.13-alpine AS development

# NOTE: This file has to be in the main folder of the wazigate-lora App 
# which is one folder up
# I use symbolic links to keep the updates
# ln -s ./main-app/Dockerfile-dev Dockerfile-dev
# ln -s ./main-app/Dockerfile Dockerfile


ENV CGO_ENABLED=0

# Copy required files to the zip folder to be compressed
COPY ./docker-compose.yml \
     ./package.json \
     /zip/
COPY ./forwarders/conf /zip/forwarders/conf
COPY ./conf /zip/conf

COPY ./app /app

WORKDIR /app


RUN apk add --no-cache ca-certificates git zip \
    && cd /zip/ \
    && zip -q -r /app/index.zip . \
    && cd /app \
    && go build -a -installsuffix cgo -ldflags "-s -w" -o wazigate-lora ./cmd/wazigate-lora

ENTRYPOINT ["tail", "-f", "/dev/null"]


#--------------------------------#


FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates curl

COPY --from=development /app/wazigate-lora .
COPY --from=development /app/index.zip /index.zip

COPY ./www/dist ./www/dist
COPY ./www/img ./www/img
COPY ./www/index.html ./www/icon.png ./www/

ENTRYPOINT ["./wazigate-lora"]