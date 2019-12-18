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

const High = true
const Low = false

const LogLevelNone = 0
const LogLevelDebug = 5
const LogLevelVerbose = 4
const LogLevelNormal = 3
const LogLevelWarning = 2
const LogLevelError = 1

var logLevel = []string{
	"[     ] ",
	"[ERR  ] ",
	"[WARN ] ",
	"",
	"[VERBO] ",
	"[DEBUG] ",
}

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
