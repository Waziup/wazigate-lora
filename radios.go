package main

import (
	"log"
	"fmt"
	"os"
	"strings"

	"github.com/Waziup/wazigate-lora/SX127X"
	"github.com/Waziup/wazigate-lora/SX1301"
	"github.com/Waziup/wazigate-lora/lora"
)

func sx1301() bool {
	SX1301.Logger = log.New(os.Stdout, "[LORA ] ", 0)
	SX1301.LogLevel = SX1301.LogLevelNormal

	radio, err := SX1301.Discover()
	if err != nil {
		logger.Printf("Err: looking for SX1301: %v", err)
		return false
	}
	err = serveRadio(radio)
	logger.Printf("Err: SX1301 stopped serving: %v", err)
	return true
}

func sx127x() bool {
	SX127X.Logger = log.New(os.Stdout, "[LORA ] ", 0)
	SX127X.LogLevel = SX127X.LogLevelNormal

	radio, err := SX127X.Discover()
	if err != nil {
		logger.Printf("Err: looking for SX127X: %v", err)
		return false
	}
	err = serveRadio(radio)
	logger.Printf("Err: SX127X stopped serving: %v", err)
	return true
}

func serveRadio(radio lora.Radio) error {

	logger.Printf("Detected radio: %s", radio.Name())

	configMutex.Lock()
	cfg := config
	configMutex.Unlock()
	
	if err := radio.Init(cfg); err != nil {
		return fmt.Errorf("can not config: %v", err)
	}

	logger.Printf("Receiving, please stand by...")


	var counter byte
	for true {

		if configChanged {
			configMutex.Lock()
			cfg = config
			configChanged = false
			configMutex.Unlock()
			if err := radio.Init(cfg); err != nil {
				return fmt.Errorf("can not config: %v", err)
			}
		}

		if len(queue) != 0 {
			msg := <-queue
			topic := strings.Split(msg.Topic, "/")
			if len(topic) == 5 {
				data := []byte{
					0x00,        // dest
					0x10 | 0x01, // type: PKT_TYPE_DATA | PKT_FLAG_DATA_DOWNLINK
					0x01,        // source
					counter,     // num
					'\\', '!',
				}
				data = append(data, 'U', 'I', 'D', '/')
				data = append(data, topic[1]...)
				data = append(data, '/')
				data = append(data, topic[3]...)
				data = append(data, '/')
				data = append(data, msg.Data...)

				if err := radio.Send(&lora.TxPacket{
					Modulation: "LORA",
					Power:      14,
					LoRaBW:     0x08,     // BW_125
					LoRaCR:     5,        // CR_5
					LoRaSF:     12,       // SF_12
					Freq:       0xD84CCC, // CH_10_868
					Data:       data,
				}); err != nil {
					logger.Printf("Err: %v", err)
				} else {
					log.Printf("[<<   ] %v", data)
				}
				counter++
			}
		}
		pkts, err := radio.Receive()
		if err != nil {
			return err
		}
		for _, pkt := range pkts {
			log.Printf("[>>   ] %s (F:%d,BW:%d,SPR:%d,CR:%d,RSSI:%d,S:%d) %v", pkt.Modulation, pkt.Freq, pkt.LoRaBW, pkt.LoRaSpr, pkt.LoRaCR, pkt.RSSI, len(pkt.Data), pkt.Data)
			process(pkt)
		}
	}
	return nil // unreachable
}
