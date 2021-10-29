// This service creates a bridge between Chirpstack and Wazigate.
// Every Wazigate Device might implement 'lorawan' metadata that will make this service create a
// a identical device in Chirpstack. The metadata should look like this:
// {
//   "lorawan": {
//      "devEUI": "AA555A0026011d87",
//      "profile": "WaziDev",
//      "devAddr": "26011d87",
//      "appSKey": "23158d3bbc31e6af670d195b5aed5525",
//      "nwkSEncKey": "d83cb057cebd2c43e21f4cde01c19ae1",
//    }
// }
package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Waziup/wazigate-lora/internal/pkg/waziapp"
	"github.com/Waziup/wazigate-lora/internal/pkg/wazigate"
	"github.com/Waziup/wazigate-lora/internal/pkg/waziup"
	asAPI "github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	asIntegr "github.com/brocaar/chirpstack-api/go/v3/as/integration"
	gw "github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

// var mqttAddr = "127.0.0.1"
// var edgeAddr = "127.0.0.1"

// var version string
// var branch string

// var gatewayID string

// Serve reads messages from the MQTT server:
// - Chirpstack AS Uplink
// - Chirpstack AS Status
// - Chirpstack AS Join
// - Chirpstack AS Ack
// - Chirpstack AS Error
// - Edge Actuator Value
// - Edge Actuator Values

func ListenAndServe() {
	handler := http.HandlerFunc(ServeHTTP)
	err := waziapp.ListenAndServe(handler)
	if err != nil {
		log.Fatal(err)
	}
}

func Serve() error {

	wazigate.Subscribe("gateway/+/event/+")
	wazigate.Subscribe("application/+/device/+/event/+")
	wazigate.Subscribe("devices/+/actuators/+/value")
	wazigate.Subscribe("devices/+/actuators/+/values")
	wazigate.Subscribe("devices/+/meta")

	for {
		msg, err := wazigate.Message()
		if err != nil {
			return err
		}
		topic := strings.Split(msg.Topic, "/")

		// Topic: devices/+/meta
		if len(topic) == 3 && topic[0] == "devices" && topic[2] == "meta" {
			// A device's metadata changed. If the device is a LoRaWAN device we will update
			// the DevEUIs map here with the DevEUI from the metadata.

			id := topic[1]
			var meta waziup.Meta
			if err = json.Unmarshal(msg.Data, &meta); err != nil {
				log.Printf("Err Can not parse device meta: %v", err)
				log.Printf("Err msg: %s", msg.Data)
				continue
			}
			checkWaziupDevice(id, meta)

			// Topic: gateway/+/event/+
		} else if len(topic) == 4 && topic[0] == "gateway" {
			// This topic is served by ChirpStack and emits Gateway events.
			// A 'gateway' from CS is just a packet forwarder for Waziup.
			switch topic[3] {
			case "stats":
				// var gwStats gw.GatewayStats
				// if err = Unmarshal(msg.Data, &gwStats); err != nil {
				// 	log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
				// 	return err
				// }
				// gwID := gwStats.GetGatewayId()
				// gwTime := gwStats.GetTime()
				// log.Printf("Forwarder %s status: %v", gwID, gwTime)
			case "up":
				var gwUp gw.UplinkFrame
				if err = proto.Unmarshal(msg.Data, &gwUp); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}

				log.Println("--- LoRaWAN Radio Rx")

				loraModInfo := gwUp.TxInfo.GetLoraModulationInfo()
				data := base64.StdEncoding.EncodeToString(gwUp.GetPhyPayload())
				gwid := binary.BigEndian.Uint64(gwUp.RxInfo.GatewayId)
				if loraModInfo != nil {
					log.Printf("Forwarder %X: LoRa: %.2f MHz, SF%d BW%d CR%s, Data: %s", gwid, float64(gwUp.TxInfo.Frequency)/1000000, loraModInfo.SpreadingFactor, loraModInfo.Bandwidth, loraModInfo.CodeRate, data)
				}
				fskModInfo := gwUp.TxInfo.GetFskModulationInfo()
				if fskModInfo != nil {
					log.Printf("Forwarder %X: FSK %.2f MHz, Bitrate: %d, Data: %s", gwid, float64(gwUp.TxInfo.Frequency)/1000000, fskModInfo.Datarate, data)
				}
			case "txack":
				log.Printf("Tx completed.")
				continue

			case "ack", "exec", "raw":
				// ignore

			default:
				log.Printf("Unknown MQTT topic %q.", msg.Topic)
				continue
			}

			// Topic: application/+/device/+/event/+
		} else if len(topic) == 6 && topic[0] == "application" && topic[2] == "device" && topic[4] == "event" {
			// This topic is served by ChirpStack and emits device data on appllication level.
			// It gives us the decrypted payload sent by a device.
			switch topic[5] {
			case "up":
				var uplinkEvt asIntegr.UplinkEvent
				if err = Unmarshal(msg.Data, &uplinkEvt); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				devEUI := binary.BigEndian.Uint64(uplinkEvt.GetDevEui())
				data := base64.StdEncoding.EncodeToString(uplinkEvt.GetData())
				log.Printf("DevEUI %X: %s", devEUI, data)

				devID := devEUIs[devEUI]
				if devID == "" {
					log.Printf("DevEUI %016X: No Waziup device for that EUI!", devEUI)
				} else {
					log.Printf("DevEUI %016X -> Waziup ID %s", devEUI, devID)
				}

				objJSON := uplinkEvt.GetObjectJson()
				if objJSON != "" {
					var lppData = make(map[string]map[string]interface{})
					err = json.Unmarshal([]byte(objJSON), &lppData)
					if err == nil {
						log.Printf("LPP Data: %+v", lppData)
						if devID != "" {
						LPPDATA:
							for sensorKind, data := range lppData {
								for channel, value := range data {
									sensorID := sensorKind + "_" + channel
									err := wazigate.AddSensorValue(devID, sensorID, value)
									if err != nil {
										if waziup.IsNotExist(err) {
											err := wazigate.AddSensor(devID, &waziup.Sensor{
												ID:    sensorID,
												Name:  sensorKind + " " + channel,
												Value: value,
												Meta: waziup.Meta{
													"createdBy": "wazigate-lora",
												},
											})
											if err != nil {
												log.Printf("Err Can not create sensor %q: %v", sensorID, err)
												continue LPPDATA
											} else {
												log.Printf("Sensor %q has been created as it did not exist.", sensorID)
											}
										} else {
											log.Printf("Err Can not create value on sensor %q: %v", sensorID, err)
										}
									}
								}
							}
						}
					} else {
						log.Printf("Err No LPP Data: %v\n%v", err, objJSON)
					}
				} else {
					log.Printf("Err The payload was not parsed by Chirpstack")
				}

			case "status":
				var statusEvt asIntegr.StatusEvent
				if err = Unmarshal(msg.Data, &statusEvt); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := statusEvt.GetDevEui()
				battery := statusEvt.GetBatteryLevel()
				log.Printf("Received status from %v: %v Battery", eui, battery)
			case "error":
				var errorEvt asIntegr.ErrorEvent
				if err = Unmarshal(msg.Data, &errorEvt); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := errorEvt.GetDevEui()
				e := errorEvt.GetError()
				log.Printf("Received error from %v: %v", eui, e)
			case "ack":
				var ackEvt asIntegr.AckEvent
				if err = Unmarshal(msg.Data, &ackEvt); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := ackEvt.GetDevEui()
				log.Printf("Received ack from %v", eui)
			case "join":
				var joinEvt asIntegr.JoinEvent
				if err = Unmarshal(msg.Data, &joinEvt); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := joinEvt.GetDevEui()
				log.Printf("Device %v joined the network.", eui)

			default:
				log.Printf("Unknown MQTT topic %q.", msg.Topic)
				continue

			}

			// Topic: devices/+/actuators/+/value
		} else if len(topic) == 5 && topic[0] == "devices" && topic[2] == "actuators" {
			// This topic is served by the Wazigate Edge and emits actuator values.
			// If the actuator belongs to a LoRaWAN device (a device with lorawan metadata)
			// then we will forward the value as payload to ChirpStack.
			devEUIInt64, ok := waziupID2devEUI(topic[1])
			if !ok {
				log.Printf("Waziup ID %s -> DevEUI ?? (no matching LoRaWAN device)", topic[1])
				continue
			}
			log.Printf("Waziup ID %s -> DevEUI %016X", topic[1], devEUIInt64)
			base64Data := base64.StdEncoding.EncodeToString([]byte(msg.Data))
			log.Printf("Payload (%s) %s", msg.Data, base64Data)
			devEUI := fmt.Sprintf("%016X", devEUIInt64)
			ctx := context.Background()
			asDeviceQueueService := asAPI.NewDeviceQueueServiceClient(chirpstack)
			resp, err := asDeviceQueueService.Enqueue(ctx, &asAPI.EnqueueDeviceQueueItemRequest{
				DeviceQueueItem: &asAPI.DeviceQueueItem{
					DevEui: devEUI,
					FPort:  100,
					Data:   []byte(base64Data),
				},
			})
			if err != nil {
				log.Printf("Can not enqueue payload: %v", err)
				continue
			}
			log.Printf("Payload enqueued: FCnt %d", resp.FCnt)

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

var devEUIs = map[uint64]string{}

func InitDevice() {

	log.Println("--- Init Device")

	for true {
		// read the current device ID (gateway ID)
		// Moved to main
		// id, err := Wazigate.GetID()
		// if err != nil {
		// 	log.Printf("Err Can not get Wazigate ID: %v", err)
		// 	log.Println("Can not call edge API, waiting for some seconds ...")
		// 	time.Sleep(3 * time.Second)
		// 	continue
		// }
		// log.Printf("Gateway ID: %s", id)

		// get all lorawan devices
		devices, err := wazigate.GetDevices(&waziup.DevicesQuery{
			Meta: []string{"lorawan"},
		})
		if err != nil {
			log.Printf("Err Can not get LoRaWAN devices: %v", err)
			log.Println("Can not call edge API, waiting for some seconds ...")
			time.Sleep(3 * time.Second)
			continue
		}

		for _, device := range devices {

			checkWaziupDevice(device.ID, device.Meta)
		}

		log.Printf("There are %d LoRaWAN devices.", len(devEUIs))

		// read device lora settings from /device/meta

		// resp, err = get("/device/meta")
		// if err != nil {
		// 	log.Fatalf("Err Can not call edge API: %s", body)
		// }

		// if resp.StatusCode != 200 {
		// 	body, _ := ioutil.ReadAll(resp.Body)
		// 	log.Printf("Err The API returned %d %s on GET \"/device/meta\".", resp.StatusCode, resp.Status)
		// 	log.Fatalf("Err Body: %s", body)
		// }

		// body, _ = ioutil.ReadAll(resp.Body)
		// var meta LoRaWANMeta
		// if err = json.Unmarshal(body, &meta); err != nil {
		// 	log.Printf("Err Can not parse device meta: %v", err)
		// 	log.Println("Err GET \"/device/meta\" returned:")
		// 	log.Fatalf("Err Body: %s", body)
		// }

		// setMeta(meta.WazigateLora)
		break
	}
}

func waziupID2devEUI(id string) (uint64, bool) {
	for devEUI, _id := range devEUIs {
		if _id == id {
			return devEUI, true
		}
	}
	return 0, false
}

func checkWaziupDevice(id string, meta waziup.Meta) error {

	lorawan := meta.Get("lorawan")
	if lorawan.Undefined() {
		return nil
	}
	devEUI, err := lorawan.Get("devEUI").String()
	if err != nil {
		log.Printf("Err Device %q DevEUI: %v", id, err)
		return nil
	}
	devEUIInt64, err := strconv.ParseUint(devEUI, 16, 64)
	if err != nil {
		log.Printf("Err Device %q DevEUI: %v", id, err)
		return nil
	}
	devEUIs[devEUIInt64] = id
	log.Printf("DevEUI %s -> Waziup ID %s", devEUI, id)
	profile, err := lorawan.Get("profile").String()
	if err != nil {
		log.Printf("Err Device %q profile: %v", id, err)
		return nil
	}
	if profile == "WaziDev" {
		if err = setDeviceProfileWaziDev(devEUI, id); err == nil {
			devAddr, err := lorawan.Get("devAddr").String()
			if err != nil {
				log.Printf("Warn Device %q not activated: devAddr: %v", id, err)
				return nil
			}
			appSKey, err := lorawan.Get("appSKey").String()
			if err != nil {
				log.Printf("Warn Device %q not activated: appSKey: %v", id, err)
				return nil
			}
			nwkSEncKey, err := lorawan.Get("nwkSEncKey").String()
			if err != nil {
				log.Printf("Warn Device %q not activated: nwkSEncKey: %v", id, err)
				return nil
			}
			setWaziDevActivation(devEUI, devAddr, nwkSEncKey, appSKey)
		}
	}
	return nil
}
