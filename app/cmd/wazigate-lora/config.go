package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	asAPI "github.com/brocaar/chirpstack-api/go/v3/as/external/api"
)

var configDirs = []string{
	".",
	"/root/app/conf/wazigate-lora",
	"/etc/wazigate-lora",
}

var configDir = "."

var config struct {
	Login          asAPI.LoginRequest    `json:"login"`
	Organization   asAPI.Organization    `json:"organization"`
	NetworkServer  asAPI.NetworkServer   `json:"network_server"`
	ServiceProfile asAPI.ServiceProfile  `json:"service_profile"`
	Gateway        asAPI.Gateway         `json:"gateway"`
	Application    asAPI.Application     `json:"application"`
	DeviceProfiles []asAPI.DeviceProfile `json:"device_profiles"`
}

func readConfig() (err error) {
	var file []byte
	for _, dir := range configDirs {
		file, err = ioutil.ReadFile(dir + "/chirpstack.json")
		if err != nil {
			err = fmt.Errorf("can not open 'chirpstack.json': %v", err)
			continue
		}
		if err = json.Unmarshal(file, &config); err != nil {
			return fmt.Errorf("can not parse 'chirpstack.json': %v", err)
		}
		configDir = dir
		return nil
	}
	return
}

func writeConfig() error {
	file, _ := json.MarshalIndent(&config, "", "  ")
	if err := ioutil.WriteFile(configDir+"/chirpstack.json", file, 0666); err != nil {
		return fmt.Errorf("can not write 'chirpstack.json': %v", err)
	}
	return nil
}
