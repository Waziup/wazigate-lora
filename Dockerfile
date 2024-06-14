FROM python:2 AS ui
# pyhton is required to build libsass for node-sass
# https://github.com/sass/node-sass/issues/3033

# libgnutls30 is required for
# https://github.com/nodesource/distributions/issues/1266
RUN apt-get update && apt-get install -y --no-install-recommends curl git libgnutls30
RUN curl -sL https://deb.nodesource.com/setup_14.x | bash -
RUN apt-get install -y --no-install-recommends nodejs

COPY www/. /ui

WORKDIR /ui/

RUN npm i && npm run build

################################################################################

FROM golang:1.19-alpine AS bin

ENV CGO_ENABLED=0

COPY . /bin

WORKDIR /bin


RUN apk add --no-cache ca-certificates git && \
    go build -a -installsuffix cgo -ldflags "-s -w" -buildvcs=false -o wazigate-lora ./cmd/wazigate-lora

#RUN apk add --no-cache ca-certificates git
#RUN go build -a -installsuffix cgo -ldflags "-s -w" -o wazigate-lora ./cmd/wazigate-lora
################################################################################


FROM alpine:latest AS app

WORKDIR /root/
RUN apk --no-cache add ca-certificates curl

# copy bin files (the wazigate-lora binary)
COPY --from=bin /bin/wazigate-lora .

# copy UI files (the wazigate-lora UI)
COPY --from=ui /ui/dist ./www/dist
COPY --from=ui /ui/img ./www/img
COPY --from=ui /ui/index.html /ui/icon.png ./www/

ENTRYPOINT ["./wazigate-lora"]
