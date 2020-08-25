#!/bin/bash
# This file initiates the LoRa packet forwarders in a fail-switch manner in order to 
# match with the hardware installed on the pi

# LoRaWAN Multi Channel
echo "\n\n============================\n"
echo "Initiating the multi-channel Lora packet forwarder..."
echo "\n============================\n\n"

echo "Restarting the CE0 pin..."
CSPIN=8
echo "$CSPIN" > /sys/class/gpio/export
sleep 1
echo "out" > /sys/class/gpio/gpio$CSPIN/direction
echo "0" > /sys/class/gpio/gpio$CSPIN/value
sleep 1
echo "1" > /sys/class/gpio/gpio$CSPIN/value
echo "Done\n"
sleep 1

echo "Restarting the Concentrator..."
RSTPIN=17
echo "$RSTPIN" > /sys/class/gpio/export
sleep 1
echo "out" > /sys/class/gpio/gpio$RSTPIN/direction
echo "1" > /sys/class/gpio/gpio$RSTPIN/value
sleep 1
echo "0" > /sys/class/gpio/gpio$RSTPIN/value
echo "Done\nLaunching the forwarder..."
sleep 2

cd /root/multi_chan_pkt_fwd/ && ./lora_pkt_fwd

#-------------------------------------------------------#

# LoRaWAN Single Channel
echo "\n\n============================\n"
echo "FAILED, switching to Single-channel Lora packet forwarder."
echo "\n============================\n\n"
cd /root/single_chan_pkt_fwd/ && ./single_chan_pkt_fwd

# In future we might need to have some sort of configuration 
# for the selection or just remove the latest option
exit 2

# # Congduc's forwarder (blocking)
# echo "\n\n============================\n"
# echo "Initiating the single-channel Congduc's Lora packet forwarder..."
# echo "\n\n============================\n\n"
# cd /root/single_congduc_pkt_fwd/ && ./single_congduc_pkt_fwd -r sx127x