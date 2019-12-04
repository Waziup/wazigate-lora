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

func (c *SX1301) On() error {
	e := C.lgw_start()
	if e != C.LGW_HAL_SUCCESS {
		return fmt.Errorf("code %d", e)
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
