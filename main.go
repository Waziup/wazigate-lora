package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	as "github.com/brocaar/chirpstack-api/go/v3/as/integration"
	gw "github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var mqttAddr = "mosquitto:1883"
var edgeAddr = "wazigate-edge"

var version string
var branch string

func main() {
	log.SetFlags(0)

	if branch != "" && version != "" {
		log.Printf("This is %s build. v%s", branch, version)
	}

	for true {
		log.Printf("Dialing \"mqtt://%s\" ...", mqttAddr)
		client, err := mqtt.Dial(mqttAddr, "wazigate-lora", true, nil, nil)
		if err != nil {
			log.Printf("Err: %v", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Printf("Waiting for messages ...")

		err = Serve(client)
		if err != nil {
			log.Printf("Err: %v", err)
			client.Disconnect()
			time.Sleep(time.Second * 5)
			continue
		}
		client.Disconnect()
	}
}

// Serve reads messages from the MQTT server:
// - Chirpstack AS Uplink
// - Chirpstack AS Status
// - Chirpstack AS Join
// - Chirpstack AS Ack
// - Chirpstack AS Error
// - Edge Actuator Value
// - Edge Actuator Values
func Serve(client *mqtt.Client) error {

	client.Subscribe("gateway/+/event/+", 0)
	client.Subscribe("application/+/device/+/+", 0)
	client.Subscribe("devices/+/actuators/+/+", 0)

	for true {
		msg, err := client.Message()
		if err != nil {
			return err
		}
		topic := strings.Split(msg.Topic, "/")
		if len(topic) == 4 && topic[0] == "gateway" {
			switch topic[3] {
			case "stats":
				var gwStats gw.GatewayStats
				if err = Unmarshal(msg.Data, &gwStats); err != nil {
					log.Printf("Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				gwID := gwStats.GetGatewayId()
				gwTime := gwStats.GetTime()
				log.Printf("Gateway %s status: %v", gwID, gwTime)
			case "up":
				var gwUp gw.UplinkFrame
				if err = Unmarshal(msg.Data, &gwUp); err != nil {
					log.Printf("Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}

				loraModInfo := gwUp.TxInfo.GetLoraModulationInfo()
				if loraModInfo != nil {
					log.Printf("Gateway %s received: %s %.2fMHz bw:%d sf:%d cr:%s", gwUp.RxInfo.GatewayId, gwUp.TxInfo.Modulation, float64(gwUp.TxInfo.Frequency)/1000000, loraModInfo.Bandwidth, loraModInfo.SpreadingFactor, loraModInfo.CodeRate)
				}
				fskModInfo := gwUp.TxInfo.GetFskModulationInfo()
				if fskModInfo != nil {
					log.Printf("Gateway %s received: %s %.2fMHz dr:%d", gwUp.RxInfo.GatewayId, gwUp.TxInfo.Modulation, float64(gwUp.TxInfo.Frequency)/1000000, fskModInfo.Datarate)
				}
			case "ack", "exec", "raw":
				// ignore

			default:
				log.Printf("Unknown MQTT topic %q.", msg.Topic)
				continue
			}
		} else if len(topic) == 5 && topic[0] == "application" && topic[2] == "device" {
			switch topic[4] {
			case "rx":
				var uplinkEvt as.UplinkEvent
				if err = Unmarshal(msg.Data, &uplinkEvt); err != nil {
					log.Printf("Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := uplinkEvt.GetDevEui()
				data := uplinkEvt.GetData()
				log.Printf("Received data from %v: %v", eui, string(data))
			case "status":
				var statusEvt as.StatusEvent
				if err = Unmarshal(msg.Data, &statusEvt); err != nil {
					log.Printf("Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := statusEvt.GetDevEui()
				battery := statusEvt.GetBatteryLevel()
				log.Printf("Received status from %v: %v Battery", eui, battery)
			case "error":
				var errorEvt as.ErrorEvent
				if err = Unmarshal(msg.Data, &errorEvt); err != nil {
					log.Printf("Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := errorEvt.GetDevEui()
				e := errorEvt.GetError()
				log.Printf("Received error from %v: %v", eui, e)
			case "ack":
				var ackEvt as.AckEvent
				if err = Unmarshal(msg.Data, &ackEvt); err != nil {
					log.Printf("Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := ackEvt.GetDevEui()
				log.Printf("Received ack from %v", eui)
			default:
				log.Printf("Unknown MQTT topic %q.", msg.Topic)
				continue

			}
		} else if len(topic) == 5 && topic[0] == "devices" && topic[2] == "actuators" {

			item := api.EnqueueDeviceQueueItemRequest{
				DeviceQueueItem: &api.DeviceQueueItem{
					DevEui: topic[3],
					FPort:  100,
					Data:   msg.Data,
				},
			}
			data, err := Marshal(&item)
			if err != nil {
				log.Printf("Can not marshal message %q: %v", msg.Topic, err)
				continue
			}
			resp, err := http.Post("/api/devices/"+topic[3]+"/queue", "application/json", bytes.NewReader(data))
			if err != nil {
				log.Printf("Can not post message %q: %v", msg.Topic, err)
				continue
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Printf("Can not post message %q: %d %s\n%s", msg.Topic, resp.StatusCode, resp.Status, body)
				continue
			}

		} else {
			log.Printf("Unknown MQTT topic %q.", msg.Topic)
			continue
		}
	}
	return nil
}

// Marshal calls protocol buffer's JSON marshaler.
func Marshal(msg proto.Message) ([]byte, error) {
	var marshaler jsonpb.Marshaler
	var buf bytes.Buffer
	err := marshaler.Marshal(&buf, msg)
	return buf.Bytes(), err
}

// Unmarshal calls protocol buffer's JSON unmarshaler.
func Unmarshal(data []byte, msg proto.Message) error {
	return jsonpb.Unmarshal(bytes.NewReader(data), msg)
}
