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

COPY ./main-app /main-app
WORKDIR /main-app


RUN apk add --no-cache ca-certificates git zip \
    && cd /zip/ \
    && zip -q -r /main-app/index.zip . \
    && cd /main-app \
    && go build -a -installsuffix cgo -ldflags "-s -w" -o main-app .

ENTRYPOINT ["tail", "-f", "/dev/null"]


#--------------------------------#


FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates curl

COPY --from=development /main-app/main-app .
COPY --from=development /main-app/index.zip /index.zip

COPY --from=development /main-app/www/dist ./www/dist
COPY --from=development /main-app/www/img ./www/img
COPY --from=development /main-app/www/index.html /main-app/www/icon.png ./www/

ENTRYPOINT ["./main-app"]