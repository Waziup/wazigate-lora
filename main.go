package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	as "github.com/brocaar/chirpstack-api/go/v3/as/integration"
	gw "github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

var mqttAddr = "127.0.0.1"
var edgeAddr = "127.0.0.1"

var version string
var branch string

var gatewayID string

func main() {
	log.SetFlags(0)

	if branch != "" && version != "" {
		log.Printf("this is %s build. v%s", branch, version)
	}

	envEdgeAddr := os.Getenv("WAZIGATE_EDGE")
	if envEdgeAddr != "" {
		edgeAddr = envEdgeAddr
	}
	log.Printf("using wazigate edge at \"http://%s\"", edgeAddr)

	go forwarder()
	initDevice()

	for true {
		if !strings.ContainsRune(mqttAddr, ':') {
			mqttAddr += ":1883"
		}
		log.Printf("dialing \"mqtt://%s\" ...", mqttAddr)
		client, err := mqtt.Dial(mqttAddr, "wazigate-lora", true, nil, nil)
		if err != nil {
			log.Printf("err: %v", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Printf("waiting for messages ...")

		err = Serve(client)
		if err != nil {
			log.Printf("err: %v", err)
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
	client.Subscribe("devices/+/actuators/+/value", 0)
	client.Subscribe("devices/+/actuators/+/values", 0)
	client.Subscribe("devices/+/meta", 0)

	for true {
		msg, err := client.Message()
		if err != nil {
			return err
		}
		topic := strings.Split(msg.Topic, "/")

		if len(topic) == 3 && topic[0] == "devices" && topic[2] == "meta" {
			id := topic[1]
			if id == gatewayID {
				var meta Meta
				if err = json.Unmarshal(msg.Data, &meta); err != nil {
					log.Printf("err Can not parse gateway meta: %v", err)
					log.Printf("err msg: %s", msg.Data)
					continue
				}
				setMeta(meta.WazigateLora)
			}

		} else if len(topic) == 4 && topic[0] == "gateway" {
			switch topic[3] {
			case "stats":
				var gwStats gw.GatewayStats
				if err = Unmarshal(msg.Data, &gwStats); err != nil {
					log.Printf("can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				gwID := gwStats.GetGatewayId()
				gwTime := gwStats.GetTime()
				log.Printf("gateway %s status: %v", gwID, gwTime)
			case "up":
				var gwUp gw.UplinkFrame
				if err = proto.Unmarshal(msg.Data, &gwUp); err != nil {
					log.Printf("err: can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				loraModInfo := gwUp.TxInfo.GetLoraModulationInfo()
				data := base64.StdEncoding.EncodeToString(gwUp.GetPhyPayload())
				gwid := binary.BigEndian.Uint64(gwUp.RxInfo.GatewayId)
				if loraModInfo != nil {
					log.Printf("gw %X: LoRa: %.2f MHz, SF%d BW%d CR%s, Data: %s", gwid, float64(gwUp.TxInfo.Frequency)/1000000, loraModInfo.SpreadingFactor, loraModInfo.Bandwidth, loraModInfo.CodeRate, data)
				}
				fskModInfo := gwUp.TxInfo.GetFskModulationInfo()
				if fskModInfo != nil {
					log.Printf("gw %X: FSK %.2f MHz, Bitrate: %d, Data: %s", gwid, float64(gwUp.TxInfo.Frequency)/1000000, fskModInfo.Datarate, data)
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
					log.Printf("can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := binary.BigEndian.Uint64(uplinkEvt.GetDevEui())
				data := base64.StdEncoding.EncodeToString(uplinkEvt.GetData())
				log.Printf("device %X: %s", eui, data)

				objJSON := uplinkEvt.GetObjectJson()
				if objJSON != "" {
					var obj = make(map[string]map[string]interface{})
					err = json.Unmarshal([]byte(objJSON), &obj)
					if err == nil {
						log.Printf("LPP data: %+v", obj)
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
						log.Printf("no LPP data: %v", err)
						log.Printf("data: %q", objJSON)
					}
				} else {
					log.Printf("the payload was not parsed")
				}

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
			case "join":
				var joinEvt as.JoinEvent
				if err = Unmarshal(msg.Data, &joinEvt); err != nil {
					log.Printf("Can not unmarshal message %q: %v", msg.Topic, err)
					return err
				}
				eui := joinEvt.GetDevEui()
				log.Printf("Device %v joined the network.", eui)

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

func initDevice() {
	for true {
		// read the current device ID (gateway ID)
		resp, err := get("/device/id")
		if err != nil {
			time.Sleep(3 * time.Second)
			log.Printf("Err %v", err)
			log.Println("Can not call edge API, waiting for some seconds ...")
			continue
		}

		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Printf("Err The API returned %d %s on GET \"/device/id\".", resp.StatusCode, resp.Status)
			log.Fatalf("Err Body: %s", body)
		}

		body, _ := ioutil.ReadAll(resp.Body)
		gatewayID = string(body)
		log.Printf("Device ID: %s", gatewayID)

		// read device lora settings from /device/meta

		resp, err = get("/device/meta")
		if err != nil {
			log.Fatalf("Err Can not call edge API: %s", body)
		}

		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Printf("Err The API returned %d %s on GET \"/device/meta\".", resp.StatusCode, resp.Status)
			log.Fatalf("Err Body: %s", body)
		}

		body, _ = ioutil.ReadAll(resp.Body)
		var meta Meta
		if err = json.Unmarshal(body, &meta); err != nil {
			log.Printf("Err Can not parse device meta: %v", err)
			log.Println("Err GET \"/device/meta\" returned:")
			log.Fatalf("Err Body: %s", body)
		}

		setMeta(meta.WazigateLora)
		break
	}
}

func get(path string) (*http.Response, error) {
	return http.Get("http://" + edgeAddr + path)
}

func post(path string, value interface{}) (*http.Response, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return http.Post("http://"+edgeAddr+path, "application/json; charset=utf-8", bytes.NewBuffer(body))
}

func isOK(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
