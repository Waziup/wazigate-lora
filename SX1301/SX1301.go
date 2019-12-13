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
	"unsafe"
	"github.com/Waziup/wazigate-rpi/gpio"
)

const High = true
const Low = false

type SX1301 struct {
	pinRst gpio.Pin
}

type BoardConf C.struct_lgw_conf_board_s
type RFConf C.struct_lgw_conf_rxrf_s
type IFConf C.struct_lgw_conf_rxif_s 

var boardconf = BoardConf{
	clksrc: 1,
	lorawan_public: true,
}

var rfconfs = []RFConf{
	RFConf{
		/* enable: */ true,
		/* freq_hz: */ 867500000,
		/* rssi_offset: */ -166.0,
		/* type: */ C.LGW_RADIO_TYPE_SX1257,
		/* tx_enable: */ false,
		/* tx_notch_freq: */ 0, // 129000,
	},
	RFConf{
		/* enable: */ true,
		/* freq_hz: */ 868500000,
		/* rssi_offset: */ -166.0,
		/* type: */ C.LGW_RADIO_TYPE_SX1257,
		/* tx_enable: */ false,
		/* tx_notch_freq: */ 0, // 129000,
	},
}

var ifconfs = []IFConf{
	IFConf{
		enable: true,
		rf_chain: 1,
		freq_hz: -400000,
	}, IFConf{
		enable: true,
		rf_chain: 1,
		freq_hz: -200000,
	}, IFConf{
		enable: true,
		rf_chain: 1,
		freq_hz: 0,
	}, IFConf{
		enable: true,
		rf_chain: 0,
		freq_hz: -400000,
	}, IFConf{
		enable: true,
		rf_chain: 0,
		freq_hz: -200000,
	}, IFConf{
		enable: true,
		rf_chain: 0,
		freq_hz: 0,
	}, IFConf{
		enable: true,
		rf_chain: 0,
		freq_hz: 200000,
	}, IFConf{
		enable: true,
		rf_chain: 0,
		freq_hz: 400000,
	},
}

var chanLoRaStd = IFConf{
	enable: true,
	rf_chain: 1,
	freq_hz: -200000,
	bandwidth: C.BW_250KHZ,
	datarate: C.DR_LORA_SF7,
}

var chanFSK = IFConf{
	enable: true,
	rf_chain: 1,
	freq_hz: 300000,
	bandwidth: C.BW_125KHZ,
	datarate: 50000,
}

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

func (c *SX1301) Off() error {
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

func (c *SX1301) Receive() ([]Packet, error) {
	pkts := make([]Packet, 16)
	e := C.lgw_receive(16, (*C.struct_lgw_pkt_rx_s)(unsafe.Pointer(&pkts[0])))
	if e == C.LGW_HAL_ERROR {
		return nil, fmt.Errorf("code %d", e)
	}
	return pkts[:e], nil
}

func delay(d int) {
	time.Sleep((time.Millisecond*time.Duration(d)))
}
