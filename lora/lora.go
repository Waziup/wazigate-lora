package lora

import "time"

// TxPacket
type TxPacket struct {
	Immediately bool // Send packet immediately (will ignore tmst & time)

	Time time.Time // Send packet on a certain timestamp value (will ignore time)
	TimeGPS  time.Time // Send packet at a certain GPS time (GPS synchronization required)

	Freq uint32 // TX central frequency in Hz

	Power int // TX output power in dBm

	ChainRF int // Concentrator "RF chain" used for TX

	Modulation string // Modulation identifier "LORA" or "FSK"

	// LoRa only
	LoRaSF byte // LoRa spreading factor
	LoRaBW byte // LoRa bandwith
	LoRaCR byte // LoRa ECC coding rate

	// FSK only
	Datarate int // FSK datarate, bits per second

	FDev int // FSK frequency deviation, in Hz

	IPol bool // Lora modulation polarization inversion
	Prea int // RF preamble size

	Data []byte // packet payload
}


// RxPacket
type RxPacket struct {
	Time time.Time // UTC time of pkt RX
	TimeGPS  time.Time // GPS time of pkt RX
	TimeFin time.Time // Internal timestamp of "RX finished" event

	Freq uint32 // RX central frequency in Hz 

	ChainIF uint8 // Concentrator "IF" channel used for RX
	ChainRF uint8 // Concentrator "RF chain" used for RX or TX

	StatCRC int8 // CRC status: 1 = OK, -1 = fail, 0 = no CRC

	Modulation string // Modulation identifier "LORA" or "FSK"

	// LoRa only
	LoRaSpr byte // LoRa spreading factor: SF7 (0x07) to SF12 (0x0c)
	LoRaBW byte // LoRa bandwith: BW7K8 (0x01), BW10K4 (0x02), BW15K6 (0x03), BW20K8 (0x04), BW31K2 (0x05), BW41K7 (0x06), BW62K5 (0x07), BW125K (0x08), BW250K (0x09), BW500K (0x0a)
	LoRaCR byte // LoRa ECC coding rate: 4/5 (0x05), 4/6 (0x06), 4/7 (0x07), 4/8 (0x08)

	// FSK only
	Datarate uint32 // FSK datarate, bits per second

	RSSI int16 // Received signal strength indication (RSSI) in dBm, 1 dB precision

	LoRaSNR float32 // Signal to noise ratio (SNR) in dB, 0.1 dB precision

	Data []byte // packet payload
}

// Radio is a LoRa chip
type Radio interface {
	// Readable radio name
	Name() string

	// Turn radio on
	Init(cfg Config) error
	// Turn radio off
	Close() error

	// Send a LoRa packet
	Send(pkt *TxPacket) error
	// Receive one or more LoRa packets
	Receive() ([]*RxPacket, error)
}

type Config struct {
	Region     string `json:"region"`
	Modulation string `json:"modulation"`
	Freq       uint32 `json:"freq"`
	CodingRate byte   `json:"codingrate"`
	Bandwidth  byte   `json:"bandwith"`
	Spreading  byte   `json:"spreading"`
	Datarate   int    `json:"datarate"`
	Power      uint8  `json:"power"`
	Preamble   uint8  `json:"preamble"`
}

