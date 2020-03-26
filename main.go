package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	asAPI "github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	asIntegr "github.com/brocaar/chirpstack-api/go/v3/as/integration"
	gw "github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

// var mqttAddr = "127.0.0.1"
// var edgeAddr = "127.0.0.1"

var version string
var branch string

var gatewayID string

func main() {
	log.SetFlags(0)

	if branch != "" && version != "" {
		log.Printf("this is %s build. v%s", branch, version)
	}

	if err := readConfig(); err != nil {
		log.Fatalf("can not read config: %v", err)
	}

	os.Remove("app/conf/socket.sock")
	listener, err := net.Listen("unix", "app/conf/socket.sock")
	if err != nil {
		log.Fatalf("can not listen on 'socket.sock': %v", err)
	}
	go http.Serve(listener, http.HandlerFunc(serveHTTP))
	defer listener.Close()

	for true {
		err := initChirpstack()
		if err == nil {
			break
		}
		log.Printf("Err %v", err)
		time.Sleep(time.Second * 5)
	}

	for true {
		initDevice()
		err := serve()
		log.Printf("Err %v", err)
		time.Sleep(time.Second * 5)
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
func serve() error {

	Wazigate.Subscribe("gateway/+/event/+")
	Wazigate.Subscribe("application/+/device/+/+")
	Wazigate.Subscribe("devices/+/actuators/+/value")
	Wazigate.Subscribe("devices/+/actuators/+/values")
	Wazigate.Subscribe("devices/+/meta")

	for true {
		msg, err := Wazigate.Message()
		if err != nil {
			return err
		}
		topic := strings.Split(msg.Topic, "/")

		// Topic: devices/+/meta
		if len(topic) == 3 && topic[0] == "devices" && topic[2] == "meta" {
			// A device's metadata changed. If the device is a LoRaWAN device we will update
			// the DevEUIs map here with the DevEUI from the metadata.

			id := topic[1]
			var meta Meta
			if err = json.Unmarshal(msg.Data, &meta); err != nil {
				log.Printf("Err Can not parse device meta: %v", err)
				log.Printf("Err msg: %s", msg.Data)
				continue
			}
			lorawan := meta.Get("lorawan")
			if !lorawan.Undefined() {
				log.Println("--- Device Meta")
				devEUIStr, err := lorawan.Get("DevEUI").String()
				if err != nil {
					log.Printf("Err Device %q DevEUI: %v", id, err)
					continue
				}
				devEUI, err := strconv.ParseUint(devEUIStr, 16, 64)
				if err != nil {
					log.Printf("Err Device %q DevEUI: %v", id, err)
					continue
				}
				devEUIs[devEUI] = id
				log.Printf("DevEUI %016X -> Waziup ID %s", devEUI, id)
			}

			// Topic: gateway/+/event/+
		} else if len(topic) == 4 && topic[0] == "gateway" {
			// This topic is served by ChirpStack and emits Gateway events.
			// A 'gateway' from CS is just a packet forwarder for Waziup.
			switch topic[3] {
			case "stats":
				var gwStats gw.GatewayStats
				if err = Unmarshal(msg.Data, &gwStats); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				gwID := gwStats.GetGatewayId()
				gwTime := gwStats.GetTime()
				log.Printf("Forwarder %s status: %v", gwID, gwTime)
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
			case "ack", "exec", "raw":
				// ignore

			default:
				log.Printf("Unknown MQTT topic %q.", msg.Topic)
				continue
			}

			// Topic: application/+/device/+/+
		} else if len(topic) == 5 && topic[0] == "application" && topic[2] == "device" {
			// This topic is served by ChirpStack and emits device data on appllication level.
			// It gives us the decrypted payload sent by a device.
			switch topic[4] {
			case "rx":
				var uplinkEvt asIntegr.UplinkEvent
				if err = Unmarshal(msg.Data, &uplinkEvt); err != nil {
					log.Printf("Err Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := binary.BigEndian.Uint64(uplinkEvt.GetDevEui())
				data := base64.StdEncoding.EncodeToString(uplinkEvt.GetData())
				log.Printf("DevEUI %X: %s", eui, data)

				dev := devEUIs[eui]
				if dev == "" {
					log.Printf("DevEUI %X: No Waziup device for that EUI!", eui)
				} else {
					log.Printf("DevEUI %X -> Waziup ID %s", eui, dev)
				}

				objJSON := uplinkEvt.GetObjectJson()
				if objJSON != "" {
					var lppData = make(map[string]map[string]interface{})
					err = json.Unmarshal([]byte(objJSON), &lppData)
					if err == nil {
						log.Printf("LPP Data: %+v", lppData)
						if dev != "" {
						LPPDATA:
							for sensorKind, data := range lppData {
								for channel, value := range data {
									sensorID := sensorKind + "_" + channel
									err := Wazigate.AddSensorValue(dev, sensorID, value)
									if err != nil {
										if IsNotExist(err) {
											err := Wazigate.AddSensor(dev, &Sensor{
												ID:    sensorID,
												Name:  sensorKind + " " + channel,
												Value: value,
												Meta: Meta{
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

						// for key, value := range values {
						// 	path := "/devices/" + eui + "/sensors/" + key + "/value"
						// 	resp, err := post(path, value)
						// 	if err != nil {
						// 		log.Printf("Err Can nor post to %s: %v", path, err)
						// 	} else if !isOK(resp.StatusCode) {
						// 		body, _ := ioutil.ReadAll(resp.Body)
						// 		log.Printf("Err Edge API %s: %d %s", path, resp.StatusCode, resp.Status)
						// 		log.Printf("%s", body)
						// 	} else {
						// 		log.Printf("Uploaded value to %s.", path)
						// 	}
						// }
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

			item := asAPI.EnqueueDeviceQueueItemRequest{
				DeviceQueueItem: &asAPI.DeviceQueueItem{
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

var devEUIs = map[uint64]string{}

func initDevice() {

	log.Println("--- Init Device")

	for true {
		// read the current device ID (gateway ID)
		id, err := Wazigate.GetID()
		if err != nil {
			log.Printf("Err Can not get Wazigate ID: %v", err)
			log.Println("Can not call edge API, waiting for some seconds ...")
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("Gateway ID: %s", id)

		// get all lorawan devices
		devices, err := Wazigate.GetDevices(&DevicesQuery{
			Meta: []string{"lorawan"},
		})
		if err != nil {
			log.Printf("Err Can not get LoRaWAN devices: %v", err)
			log.Println("Can not call edge API, waiting for some seconds ...")
			time.Sleep(3 * time.Second)
			continue
		}

		for _, device := range devices {
			meta := device.Meta.Get("lorawan")
			if meta.Undefined() {
				log.Printf("Err Device %s has no lorawan meta?!?!", device.ID)
				continue
			}
			devEUIStr, err := meta.Get("DevEUI").String()
			if err != nil {
				log.Printf("Err Device %s DevEUI:", err)
				continue
			}
			devEUI, err := strconv.ParseUint(devEUIStr, 16, 64)
			if err != nil {
				log.Printf("Err Device %s DevEUI:", err)
				continue
			}
			devEUIs[devEUI] = device.ID
			log.Printf("DevEUI %016X -> Waziup ID %s", devEUI, device.ID)
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
