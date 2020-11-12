FROM golang:buster AS development

# We need to use Buster as Alpine removed `qsort` lib that is used by the forwarder

# Building the multi chan packet forwarder for RAK2245
RUN git clone https://github.com/Lora-net/lora_gateway \
    && git clone https://github.com/Lora-net/packet_forwarder multi_chan_pkt_fwd \
    # lowering the SPI speed for more reliability
    && sed -i -- 's/SPI_SPEED.*8000000/SPI_SPEED 1000000/g' ./lora_gateway/libloragw/src/loragw_spi.native.c \
    && cd multi_chan_pkt_fwd \
    && ./compile.sh \
    && mkdir -p /build \
    && mkdir -p /build/multi_chan_pkt_fwd \
    && cp ./lora_pkt_fwd/lora_pkt_fwd /build/multi_chan_pkt_fwd \
    && cp ./lora_pkt_fwd/*.json /build/multi_chan_pkt_fwd

# Building the single chan packet forwarder for WAziHat
RUN git clone https://github.com/Waziup/single_chan_pkt_fwd single_chan_pkt_fwd \
    && cd single_chan_pkt_fwd \
    && go build -o build/single_chan_pkt_fwd . \
    && mkdir -p /build/single_chan_pkt_fwd \
    && cp -r ./build/* /build/single_chan_pkt_fwd \
    && cp ./*.json /build/single_chan_pkt_fwd

# Building the single Congduc packet forwared for SX1301
RUN git clone -b master https://github.com/Waziup/wazigate-lora single_congduc_pkt_fwd \
    && cd single_congduc_pkt_fwd \
    && cp ../lora_gateway/libloragw/libloragw.a SX1301/libs \
    && mkdir -p /build/single_congduc_pkt_fwd \
    && export branch=$(git rev-parse --abbrev-ref HEAD) \
    && export version=$(git describe --always) \
    && go build -ldflags "-s -w -X main.version=$version -X main.branch=$branch" -o build/single_congduc_pkt_fwd . \
    && cp -r ./build/* /build/single_congduc_pkt_fwd \
    && cp ./*.json /build/single_congduc_pkt_fwd

# Building the multi chan packet forwarder for USB HT-M01
RUN git clone https://github.com/Lora-net/picoGW_hal.git \
    && git clone https://github.com/Lora-net/picoGW_packet_forwarder.git \
    && cd picoGW_hal \
    && make clean all \
    && cd ../picoGW_packet_forwarder \
    && make clean all \
    && mkdir -p /build/picoGW_pkt_fwd \
    && cp ./lora_pkt_fwd/lora_pkt_fwd /build/picoGW_pkt_fwd \ 
    && cp ./lora_pkt_fwd/*.json /build/picoGW_pkt_fwd \
    && echo "test"

# Building the multi chan packet forwarder for USB HT-M01
RUN git clone https://github.com/Lora-net/picoGW_hal.git \
    && cd picoGW_hal \
    && make \
    && cd .. \
    && git clone https://github.com/Lora-net/picoGW_packet_forwarder.git \
    && cd picoGW_packet_forwarder \
    && make 


#--------------------------------#

FROM debian:buster-slim AS production

RUN apt-get update \
    && apt-get install -y curl

WORKDIR /root/

# Copying over all executables
COPY --from=development /opt/waziup/packet_forwarder/lora_pkt_fwd/lora_pkt_fwd        spi_multi_chan/lora_pkt_fwd
COPY --from=development /opt/waziup/single_chan_pkt_fwd/build/single_chan_pkt_fwd     single_chan/lora_pkt_fwd
COPY --from=development /opt/waziup/wazigate-lora/build/single_congduc_pkt_fwd        single_congduc/lora_pkt_fwd
COPY --from=development /opt/waziup/picoGW_packet_forwarder/lora_pkt_fwd/lora_pkt_fwd usb_multi_chan/lora_pkt_fwd

COPY ./start.sh .

ENTRYPOINT ["bash", "./start.sh"]
