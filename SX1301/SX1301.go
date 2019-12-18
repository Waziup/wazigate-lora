package SX1301

// #cgo CFLAGS: -O2 -Wall -Wextra -Wno-unused-parameter -std=c99 -Iinc
// #cgo LDFLAGS: -Llibs -lloragw -lrt -lm
//
// #if __STDC_VERSION__ >= 199901L
//     #define _XOPEN_SOURCE 600
// #else
//     #define _XOPEN_SOURCE 500
// #endif
//
// #include "loragw_hal.h"
import "C"

import (
	"time"
	"fmt"
	"log"
	"unsafe"
	"github.com/Waziup/wazigate-rpi/gpio"
	"github.com/Waziup/wazigate-lora/lora"
)



var LogLevel = LogLevelNone
var Logger *log.Logger

type SX1301 struct {
	pinRst gpio.Pin

	LogLevel int
	Logger *log.Logger
}

func (c *SX1301) Name() string {
	return "SX1301"
}

type BoardConf C.struct_lgw_conf_board_s
type RFConf C.struct_lgw_conf_rxrf_s
type IFConf C.struct_lgw_conf_rxif_s 

func New(pinRst gpio.Pin) *SX1301 {
	return &SX1301{
		pinRst: pinRst,
	}
}

func (c *SX1301) Reset() error {
	c.pinRst.Write(Low)
	delay(100)
	c.pinRst.Write(High)
	delay(100)
	return nil
}

// var lwgm uint64 = 0xAA555A0000000000

func Discover() (lora.Radio, error) {

	// Reset GIOP Pin
	pinRst, err := gpio.Output(17)
	if err != nil {
		return nil, err
	}

	// SX3101 instance
	c := New(pinRst)
	c.Logger = Logger
	c.LogLevel = LogLevel
		
	// Startup ...
	if err = c.On(); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil;
}

func (c *SX1301) Init(cfg lora.Config) error {
	return nil
}


func (c *SX1301) On() error {
	e := C.lgw_board_setconf(C.struct_lgw_conf_board_s(boardconf))
	if e != C.LGW_HAL_SUCCESS {
		return fmt.Errorf("can not set board conf: code %d", e)
	}

	for i, rfconf := range rfconfs {
		e = C.lgw_rxrf_setconf(C.uchar(i), C.struct_lgw_conf_rxrf_s(rfconf))
		if e != C.LGW_HAL_SUCCESS {
			return fmt.Errorf("can not set rxrf conf: code %d", e)
		}
	}

	for i, ifconf := range ifconfs {
		e = C.lgw_rxif_setconf(C.uchar(i), C.struct_lgw_conf_rxif_s(ifconf))
		if e != C.LGW_HAL_SUCCESS {
			return fmt.Errorf("can not set rxif conf: code %d", e)
		}
	}

	e = C.lgw_rxif_setconf(8, C.struct_lgw_conf_rxif_s(chanLoRaStd))
	if e != C.LGW_HAL_SUCCESS {
		return fmt.Errorf("can not set rxif conf (LoRa std): code %d", e)
	}

	e = C.lgw_rxif_setconf(9, C.struct_lgw_conf_rxif_s(chanFSK))
	if e != C.LGW_HAL_SUCCESS {
		return fmt.Errorf("can not set rxif conf (LoRa fsk): code %d", e)
	}

	e = C.lgw_start()
	if e != C.LGW_HAL_SUCCESS {
		return fmt.Errorf("can not turn on: code %d", e)
	}

	return nil
}

func (c *SX1301) Close() error {
	e := C.lgw_stop()
	if e != C.LGW_HAL_SUCCESS {
		return fmt.Errorf("code %d", e)
	}
	return nil
}

/*
type PacketX struct {
	freq_hz uint32     
    if_chain uint8    
    status uint8     
    count_us uint32 
    rf_chain uint8    
    modulation uint8   
    bandwidth uint8    
    datarate uint32    
    coderate uint8     
    rssi float32   
    snr float32            
    snr_min float32        
    snr_max float32        
    crc uint16            
    size uint16           
    payload [256]uint8   
}
*/

type Packet C.struct_lgw_pkt_rx_s


func (c *SX1301) Receive() ([]*lora.RxPacket, error) {
	cpkts := make([]Packet, 16)
	e := C.lgw_receive(16, (*C.struct_lgw_pkt_rx_s)(unsafe.Pointer(&cpkts[0])))
	if e == C.LGW_HAL_ERROR {
		return nil, fmt.Errorf("code %d", e)
	}
	if e == 0 {
		return nil, nil
	}
	pkts := make([]*lora.RxPacket, e)
	for i := 0; i<int(e); i++ {
		cpkt := cpkts[i]
		pkt := &lora.RxPacket{
			//Time: 1,
			ChainIF: uint8(cpkt.if_chain),
			ChainRF: uint8(cpkt.rf_chain),
			Freq: uint32(cpkt.freq_hz),
		}

		switch cpkt.status {
		case C.STAT_CRC_OK:
			pkt.StatCRC = 1
		case C.STAT_CRC_BAD:
			pkt.StatCRC = -1
		case C.STAT_NO_CRC:
			pkt.StatCRC = 0
		}

		if cpkt.modulation == C.MOD_LORA {
			pkt.Modulation = "LORA"
			pkt.Freq = uint32(cpkt.freq_hz)
			pkt.RSSI = int16(cpkt.rssi)
			pkt.LoRaBW = byte(cpkt.bandwidth)
			switch cpkt.bandwidth {
			case C.BW_7K8HZ:
				pkt.LoRaBW = 1
			case C.BW_15K6HZ:
				pkt.LoRaBW = 3
			case C.BW_31K2HZ:
				pkt.LoRaBW = 5
			case C.BW_62K5HZ:
				pkt.LoRaBW = 7
			case C.BW_125KHZ:
				pkt.LoRaBW = 8
			case C.BW_250KHZ:
				pkt.LoRaBW = 9
			case C.BW_500KHZ:
				pkt.LoRaBW = 10

			}
			switch cpkt.coderate {
			case C.CR_LORA_4_5:
				pkt.LoRaCR = 5
			case C.CR_LORA_4_6:
				pkt.LoRaCR = 6
			case C.CR_LORA_4_7:
				pkt.LoRaCR = 7
			case C.CR_LORA_4_8:
				pkt.LoRaCR = 8
			}
			switch cpkt.datarate {
			case C.DR_LORA_SF7:
				pkt.LoRaSpr = 7
			case C.DR_LORA_SF8:
				pkt.LoRaSpr = 8
			case C.DR_LORA_SF9:
				pkt.LoRaSpr = 9
			case C.DR_LORA_SF10:
				pkt.LoRaSpr = 10
			case C.DR_LORA_SF11:
				pkt.LoRaSpr = 11
			case C.DR_LORA_SF12:
				pkt.LoRaSpr = 12
			}
		}
		if cpkt.modulation == C.MOD_FSK {
			pkt.Modulation = "FSK"
			pkt.Datarate = uint32(cpkt.datarate)
		}

		pkt.Data = make([]byte, cpkt.size)
		for j:=0; j<len(pkt.Data); j++ {
			pkt.Data[j] = byte(cpkt.payload[j])
		}

		pkts[i] = pkt
	}
	return pkts, nil
}

func delay(d int) {
	time.Sleep((time.Millisecond*time.Duration(d)))
}

func (c *SX1301) Send(pkt *lora.TxPacket) (err error) {

	return nil
}
