package wazigate

import (
	"os"
	"path"

	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-lora/internal/pkg/waziup"
)

var defaultEdgeHost = "wazigate-edge"

func getEdgeHost() string {
	edgeHost := os.Getenv("WAZIGATE_EDGE")
	if edgeHost == "" {
		return defaultEdgeHost
	}
	return edgeHost
}

var conn *waziup.Waziup

var Dir = "/var/lib/wazigate"

func AppsDir() string {
	return path.Join(Dir, "apps")
}

func AppDir(name string) string {
	return path.Join(AppsDir(), name)
}

func Connect() (err error) {
	conn, err = waziup.Connect(&waziup.ConnectSettings{
		Host: getEdgeHost(),
	})
	return err
}

func ID() (id string, err error) {
	return conn.GetID()
}

func Subscribe(topic string) (err error) {
	return conn.Subscribe(topic)
}

func Message() (*mqtt.Message, error) {
	return conn.Message()
}

func AddSensorValue(deviceID string, sensorID string, value interface{}) error {
	return conn.AddSensorValue(deviceID, sensorID, value)
}

func AddSensor(deviceID string, sensor *waziup.Sensor) error {
	return conn.AddSensor(deviceID, sensor)
}

func GetDevices(query *waziup.DevicesQuery) (devices []waziup.Device, err error) {
	return conn.GetDevices(query)
}
