#!/bin/bash
# This file initiates the LoRa packet forwarders in a fail-switch manner in order to 
# match with the hardware installed on the pi

# function definitions

# reset the concentrator via GPIO pins
function reset_gpio {

  echo -e "Restarting the CE0 pin..."
  CSPIN=8
  echo "$CSPIN" > /sys/class/gpio/export
  sleep 1
  echo "out" > /sys/class/gpio/gpio$CSPIN/direction
  echo "0" > /sys/class/gpio/gpio$CSPIN/value
  sleep 1
  echo "1" > /sys/class/gpio/gpio$CSPIN/value
  echo -e "Done\n"
  sleep 1
  
  echo -e "Restarting the Concentrator..."
  RSTPIN=17
  echo "$RSTPIN" > /sys/class/gpio/export
  sleep 1
  echo "out" > /sys/class/gpio/gpio$RSTPIN/direction
  echo "1" > /sys/class/gpio/gpio$RSTPIN/value
  sleep 1
  echo "0" > /sys/class/gpio/gpio$RSTPIN/value
  sleep 1
}

# Start the forwarder
# parameter 1: location of the forwarder executable
# parameter 2: location of the config file (global_conf.json)
function start_forwarder {
    cd $(dirname "$1")
    # Linking global_conf.json
    ln -s $2
    ls
    # Create the local_conf.json with our gateway ID
    echo "{\"gateway_conf\": {\"gateway_ID\": \"${GWID}\"}}" >> local_conf.json
    # launch the concentrator
    ./$(basename "$1")
}


GWID=$(curl -s http://wazigate-edge/device/id | tr -d '"')
echo -e "Gateway ID is: ${GWID}"

# Trying all concentrators, one at a time

echo -e "Initiating the SPI multi-channel Lora packet forwarder..."
reset_gpio
start_forwarder ~/spi_multi_chan/lora_pkt_fwd ~/conf/multi_chan_pkt_fwd/global_conf.json

echo -e "Restarting the RFM95x Module..."
RSTPIN=17
echo "$RSTPIN" > /sys/class/gpio/export
sleep 1
echo "out" > /sys/class/gpio/gpio$RSTPIN/direction
echo "0" > /sys/class/gpio/gpio$RSTPIN/value
sleep 1
echo "1" > /sys/class/gpio/gpio$RSTPIN/value
echo -e "Done\nLaunching the forwarder..."
sleep 1

echo -e "Initiating the single-channel Lora packet forwarder..."
reset_gpio
start_forwarder ~/single_chan/lora_pkt_fwd ~/conf/single_chan_pkt_fwd/global_conf.json

# In future we might need to have some sort of configuration 
# for the selection or just remove the latest option

# # Congduc's forwarder (blocking)
# echo -e "\n\n============================\n"
# echo -e "Initiating the single-channel Congduc's Lora packet forwarder..."
# echo -e "\n\n============================\n\n"
# cd /root/single_congduc_pkt_fwd/ && ./single_congduc_pkt_fwd -r sx127x


#---------------------------------------------#

# LoRaWAN HT-M01
echo -e "\n\n============================\n"
echo -e "Initiating the HT-M01 Lora packet forwarder..."
echo -e "\n============================\n\n"

cd /root/picoGW_pkt_fwd/
./lora_pkt_fwd

echo -e "All forwarders failed, exiting." 
exit 2
