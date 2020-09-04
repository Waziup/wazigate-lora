package main

import "log"

// Meta is device metadata.
type LoRaWANMeta struct {
	WazigateLora *WazigateLora `json:"lorawan"`
}

type WazigateLora struct {
	Region    string `json:"region"`
	Frequency int    `json:"frequency"`
	Forwarder string `json:"forwarder"`
	Spreading int    `json:"spreading"`
}

func setMeta(lora *WazigateLora) {

	if lora == nil {
		log.Println("The device has no LoRa settings (\"wazigate-lora\" in meta).")
		log.Println("The LoRa radio will be will be halted.")
		setForwarder(noForwarder)
		return
	}

	log.Printf("LoRa: %+v", lora)

	if lora.Forwarder == "" {
		log.Println("The device has no forwader set (\"wazigate-lora.fwd\" in meta).")
		log.Println("The LoRa radio will be will be halted.")
		setForwarder(noForwarder)
		return
	}

	var fwd Forwarder
	fwd.config = "forwader/" + lora.Forwarder + "_" + lora.Region + "_global_config.json"
	fwd.exec = "forwader/" + lora.Forwarder
	setForwarder(fwd)
}
