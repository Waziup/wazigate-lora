package app

import (
	"github.com/Waziup/wazigate-lora/internal/pkg/waziapp"
	asAPI "github.com/chirpstack/chirpstack/api/go/v4/api"
)

var Config struct {
	Login          asAPI.LoginRequest    `json:"login"`
	Tenant         asAPI.Tenant          `json:"tenant"`
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
