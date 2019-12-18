package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

    "github.com/Waziup/wazigate-lora/lora"
)

var config = lora.Config{
	Region:     "EU",
	Modulation: "LORA",
	Freq:       865200000, // CH_10_868
	Bandwidth:  0x08,      // BW125K
	CodingRate: 5,         // 4/5
	Spreading:  12,        // SF12
	Power:      14,        // dBm
	Preamble:   8,         // symbols
	// Datarate: ,
}

var configMutex sync.Mutex
var configChanged = false

type Meta struct {
	LoraMeta *lora.Config `json:"wazigate-lora"`
}

type Device struct {
	Meta Meta `json:"meta"`
}

func fetchConfig() (bool, error) {

	resp, err := http.Get("http://127.0.0.1:880/device")
	if err != nil {
		return false, fmt.Errorf("failed to call Edge API: %v", err)
	} else if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("http response %d: %s", resp.StatusCode, body)
	} else {
		decoder := json.NewDecoder(resp.Body)
		var device Device
		err = decoder.Decode(&device)
		if err != nil {
			return false, fmt.Errorf("failed to read Edge API: %v", err)
		}

		if device.Meta.LoraMeta != nil {
			return applyConfig(*device.Meta.LoraMeta)
		}
		return false, nil
	}
}

func applyConfig(nc lora.Config) (bool, error) {
	oc := config
	changed := false

	if nc.Modulation != "" && nc.Modulation != oc.Modulation {
		if nc.Modulation != "LORA" && nc.Modulation != "FSK" {
			return false, fmt.Errorf("invalid modulation %q: must be \"LORA\" or \"FSK\"", nc.Modulation)
		}
		changed = true
		oc.Modulation = nc.Modulation
	}
	if nc.Power != 0 && nc.Power != oc.Power {
		changed = true
		oc.Power = nc.Power
	}

	if nc.Freq != 0 && nc.Freq != oc.Freq {
		changed = true
		oc.Freq = nc.Freq
	}

	if oc.Modulation == "LORA" {
		if nc.Bandwidth != 0 && nc.Bandwidth != oc.Bandwidth {
			if nc.Bandwidth < 1 ||  nc.Bandwidth > 10 {
				return false, fmt.Errorf("invalid bandwidth %d: must be 1~10", nc.Bandwidth)
			}
			changed = true
			oc.Bandwidth = nc.Bandwidth
		}
		if nc.Spreading != 0 && nc.Spreading != oc.Spreading {
			if nc.Spreading < 7 ||  nc.Spreading > 12 {
				return false, fmt.Errorf("invalid spreading %d: must be 7~12", nc.Spreading)
			}
			changed = true
			oc.Spreading = nc.Spreading
		}
		if nc.CodingRate != 0 && nc.CodingRate != oc.CodingRate {
			if nc.CodingRate < 5 ||  nc.CodingRate > 8 {
				return false, fmt.Errorf("invalid coding rate %d: must be 5~8", nc.CodingRate)
			}
			changed = true
			oc.CodingRate = nc.CodingRate
		}
	} else if oc.Modulation == "FSK" {
		if nc.Datarate != 0 && nc.Datarate != oc.Datarate {
			changed = true
			oc.Datarate = nc.Datarate
		}
	}

	if changed {	
		configMutex.Lock()
		config = oc
		configChanged = true
		configMutex.Unlock()
		printConfig()
	}

	return changed, nil
}

func printConfig() {
	if config.Modulation == "LORA" {
		logger.Printf("LORA: Region: %s, Freq: %.2fMHz, BW: %d, SF: %d, CR: %d, Power: %ddBm", config.Region, float32(config.Freq)/1e6, config.Bandwidth, config.Spreading, config.CodingRate, config.Power)
	} else /* if config.Modulation == "FSK" */ {
		logger.Printf("FSK: Region: %s, Freq: %.2fMHz, DR: %d, Power: %ddBm", config.Region, float32(config.Freq)/1e6, config.Datarate, config.Power)
	}
}
