FROM golang:buster AS development
# We need to use Buster as Alpine removed `qsort` lib that is used by the forwarder

WORKDIR /opt/waziup

# Building the multi chan packet forwarder for RAK2245
RUN git clone https://github.com/Lora-net/lora_gateway \
    # lowering the SPI speed for more reliability
    && sed -i -- 's/SPI_SPEED.*8000000/SPI_SPEED 2000000/g' ./lora_gateway/libloragw/src/loragw_spi.native.c \
    && git clone https://github.com/Lora-net/packet_forwarder \
    && cd packet_forwarder \
    && ./compile.sh

# Building the single chan packet forwarder for WAziHat
RUN git clone https://github.com/Waziup/single_chan_pkt_fwd \
    && cd single_chan_pkt_fwd \
    && go build -o build/single_chan_pkt_fwd .

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
COPY --from=development /opt/waziup/lora_gateway/reset_lgw.sh                         spi_multi_chan/reset_lgw.sh
COPY --from=development /opt/waziup/single_chan_pkt_fwd/build/single_chan_pkt_fwd     single_chan/lora_pkt_fwd
COPY --from=development /opt/waziup/picoGW_packet_forwarder/lora_pkt_fwd/lora_pkt_fwd usb_multi_chan/lora_pkt_fwd

COPY ./start.sh .

RUN ls

# Uncomment For development
#ENTRYPOINT ["tail", "-f", "/dev/null"]

# Uncomment For production
ENTRYPOINT ["bash", "./start.sh"]
