package lora

import "time"

// TxPacket
type TxPacket struct {
	Immediately bool // Send packet immediately (will ignore tmst & time)

	Time time.Time // Send packet on a certain timestamp value (will ignore time)
	TimeGPS  time.Time // Send packet at a certain GPS time (GPS synchronization required)

	Freq float32// TX central frequency in MHz

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
	Size int // RF packet payload size in bytes

	Data []byte // packet payload
}


// RxPacket
type RxPacket struct {
	Time time.Time // UTC time of pkt RX
	TimeGPS  time.Time // GPS time of pkt RX
	TimeFin time.Time // Internal timestamp of "RX finished" event

	Freq float32// RX central frequency in MHz 

	ChanIF int // Concentrator "IF" channel used for RX
	ChainRF int // Concentrator "RF chain" used for RX or TX

	StatCRC byte // CRC status: 1 = OK, -1 = fail, 0 = no CRC

	Modulation string // Modulation identifier "LORA" or "FSK"

	// LoRa only
	LoRaSpr byte // LoRa sSpreading factor
	LoRaBW byte // LoRa bandwith
	LoRaCR byte // LoRa ECC coding rate

	// FSK only
	Datarate int // FSK datarate, bits per second

	RSSI int16 // RSSI in dBm, 1 dB precision

	LoRaSNR float32 // Lora SNR ratio in dB, 0.1 dB precision

	Data []byte // packet payload
}

// Radio is a LoRa chip
type Radio interface {
	// Readable radio name
	Name() string

	// Turn radio on
	On() error
	// Turn radio off
	Off() error

	// Send a LoRa packet
	Send(pkt *TxPacket) error
	// Receive one or more LoRa packets
	Receive() ([]*RxPacket, error)
}

