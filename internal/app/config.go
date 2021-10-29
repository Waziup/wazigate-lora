package app

import (
	"github.com/Waziup/wazigate-lora/internal/pkg/waziapp"
	asAPI "github.com/brocaar/chirpstack-api/go/v3/as/external/api"
)

var Config struct {
	Login          asAPI.LoginRequest    `json:"login"`
	Organization   asAPI.Organization    `json:"organization"`
	NetworkServer  asAPI.NetworkServer   `json:"network_server"`
	ServiceProfile asAPI.ServiceProfile  `json:"service_profile"`
	Gateway        asAPI.Gateway         `json:"gateway"`
	Application    asAPI.Application     `json:"application"`
	DeviceProfiles []asAPI.DeviceProfile `json:"device_profiles"`
}

func ReadConfig() (err error) {
	return waziapp.ReadConfig(&Config)
}

func WriteConfig() error {
	return waziapp.WriteConfig(&Config)
}
