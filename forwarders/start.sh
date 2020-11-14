#!/bin/bash
# This file initiates the LoRa packet forwarders in a fail-switch manner in order to 
# match with the hardware installed on the pi

echo -e "Setting the Gateway ID..."
GWID=$(curl -s http://wazigate-edge/device/id | tr -d '"')
echo -e "Gateway ID is: ${GWID}"

cfgFiles=("/root/multi_chan_pkt_fwd/global_conf.json" "/root/multi_chan_pkt_fwd/local_conf.json" "/root/single_chan_pkt_fwd/global_conf.json" "/root/single_chan_pkt_fwd/local_conf.json")

#mkdir -p tmp
#for f in "${cfgFiles[@]}"
#do
#    rm -f ./tmp/*
#    cp $f ./tmp/test.json
#    sed -i 's/\(^\s*"gateway_ID":\s*"\).*"\s*\(,\?\).*$/\1'${GWID}'"\2/' ./tmp/test.json
#    cp -f ./tmp/test.json $f
#    echo -e "[ $f ]" "Done"
#done

pwd
find

#---------------------------------------------#


# LoRaWAN Multi Channel
echo -e "\n\n============================\n"
echo -e "Initiating the multi-channel Lora packet forwarder..."
echo -e "\n============================\n\n"

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
echo -e "Done\nLaunching the forwarder..."
sleep 2

cd spi_multi_chan/
ln -s ~/conf/multi_chan_pkt_fwd/global_conf.json
ln -s ~/conf/multi_chan_pkt_fwd/local_conf.json
ls -la
./lora_pkt_fwd

#-------------------------------------------------------#

# LoRaWAN Single Channel
echo -e "\n\n============================\n"
echo -e "FAILED, switching to Single-channel Lora packet forwarder."
echo -e "\n============================\n\n"


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

cd single_chan/
ln -s ~/conf/single_chan_pkt_fwd/global_conf.json
./lora_pkt_fwd

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

cd usb_multi_chan/
ln -s ~/conf/multi_chan_pkt_fwd/global_conf.json
ln -s ~/conf/multi_chan_pkt_fwd/local_conf.json
./lora_pkt_fwd

echo -e "All forwarders failed, exiting." 
exit 2
