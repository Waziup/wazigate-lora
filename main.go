package main

import (
	"log"
	"strconv"
	"time"
	"fmt"
	"bytes"
	"io/ioutil"
	"os"
	"errors"
	"encoding/json"
	"strings"
	"net/http"
    "github.com/Waziup/wazigate-lora/SX127X"
    "github.com/Waziup/wazigate-lora/SX1301"
    "github.com/Waziup/wazigate-lora/lora"
    "github.com/Waziup/wazigate-rpi/gpio"
	"github.com/Waziup/wazigate-edge/mqtt"
)

const EdgeOrigin = "http://127.0.0.1:880"
const ContentType = "application/json; charset=utf-8"
const Radios = "sx127x,sx1272,sx1276,sx1301"

var queue = make(chan *mqtt.Message, 8)
var offline = false

func sx1301() bool {
	pinRst, err := gpio.Output(17)
	if err != nil {
        return false
	}
	sx := SX1301.New(pinRst)
	err = sx.On()
	log.Println("[     ] Chip SX1301 detected.")
	if err != nil {
		return false
	}
	defer sx.Off()
	for true {
		pkts, err := sx.Receive()
		if err != nil {
			return false
		}
		if len(pkts) == 0 {
			time.Sleep(200*time.Millisecond)
			continue
		}
		logger.Printf("%#v", pkts)
	}
	return true
}

func sx127x() bool {
	SX127X.Logger = log.New(os.Stdout, "[LORA ] ", 0)
	SX127X.LogLevel = SX127X.LogLevelNormal

	radio, err := SX127X.Discover()
	if err != nil {
		logger.Printf("Err: looking for SX127X: %v", err)
		return false
	}
	logger.Printf("Detected radio: %s", radio.Name())
	logger.Printf("Receiving, please stand by...")
	err = serveRadio(radio)
	logger.Printf("Err: SX127X stopped serving: %v", err)
	return true
}

func serveRadio(radio lora.Radio) error {
	var counter byte
	for true {

		if len(queue) != 0 {
			msg := <- queue
			topic := strings.Split(msg.Topic, "/")
			if len(topic) == 5 {
				data := []byte{
					0x00, // dest
					0x10|0x01, // type: PKT_TYPE_DATA | PKT_FLAG_DATA_DOWNLINK
					0x01, // source
					counter, // num
					'\\', '!',
				}
				data = append(data, 'U', 'I', 'D', '/')
				data = append(data, topic[1]...)
				data = append(data, '/')
				data = append(data, topic[3]...)
				data = append(data, '/')
				data = append(data, msg.Data...)

				if err := radio.Send(&lora.TxPacket{
					Modulation: "LORA",
					Power: 14,
					LoRaBW: 7, // BW_125
					LoRaCR: 1, // CR_5
					LoRaSF: 12, // SF_12
					Freq: 0xD84CCC, // CH_10_868
					Data: data,
				}); err != nil {
					logger.Printf("Err: %v", err)
				} else {
					log.Printf("[<<   ] %v", data)
				}
				counter++
			}
		}
		pkts, err := radio.Receive()
		if err != nil {
			return err
		}
		for _, pkt := range pkts {
			log.Printf("[>>   ] %v (rssi: %d)", pkt.Data, pkt.RSSI)
			if (len(pkt.Data) > 6 && pkt.Data[4] == '\\' && pkt.Data[5] == '!') {
				dest := pkt.Data[0]
				typ := pkt.Data[1]
				src := pkt.Data[2]
				num := pkt.Data[3]
				logger.Printf("Dest: %d, Typ: %d, Src: %d, Num: %d, Data: %q, RSSI: %d", dest, typ, src, num, pkt.Data[6:], pkt.RSSI)
				err = PostData(int(src), pkt.Data[6:])
				if err != nil {
					logger.Printf("Err: can not save data: %v", err)
				}
			} else {
				// LoRaWAN:
				// PHYPayload: MHDR | MACPayload | MIC
				// MHDR: MType | RFU | Major
				logger.Printf("Err: unknown receive format")
			}
		}
	}
	return nil // unreachable
}

var logger *log.Logger

func main() {
	log.SetFlags(0)
	logger = log.New(os.Stdout, "[     ] ", 0)

	radios := Radios
	for i := 1; i<len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-r", "-radios":
			i++
			if i == len(os.Args) {
				logger.Fatalf("Err: argument %q is missing its value", arg)
			}
			radios = strings.ToLower(os.Args[i])
			if !strings.Contains(radios, "sx127x") && !strings.Contains(radios, "sx1301") {
				logger.Fatalf("none of the radios match %q.", Radios)
			}

		case "-o", "-offline":
			offline = true
		default:
			logger.Fatalf("Err: unrecognized argument: %q", arg)
		}
	}

	if (!offline) {
		id, err := GetLocalID()
		if err != nil {
			logger.Printf("Err: can not connect to local edge service: %v", err)
		} else {
			logger.Printf("Edge ID: %q", id)
		}
		id = ""
		err = nil
		go downstream()
	}

	logger.Println("Looking for connected radios...")

	for true {
		if strings.Contains(radios, "sx127x") {
			if sx127x() {
				time.Sleep(time.Second/2)
				continue
			}
			time.Sleep(time.Second/2)
		}
		if strings.Contains(radios, "sx1301") {
			if sx1301() {
				time.Sleep(time.Second/2)
				continue
			}
			time.Sleep(time.Second/2)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////


func downstream() {
	logger := log.New(os.Stdout, "[MQTT ] ", 0)
	for true {
		client, err := mqtt.Dial("127.0.0.1:1883", "wazigate-lora", true, nil, nil)
		if err != nil {
			logger.Printf("Err: %v", err)
			time.Sleep(time.Second*2)
			continue
		}
		logger.Println("Connected to edge mqtt.")
		client.Subscribe("devices/+/actuators/+/value", 0)
		for true {
			msg, err := client.Message()
			if err != nil {
				logger.Printf("Err: %v", err)
				time.Sleep(time.Second*2)
				break
			}
			if msg == nil {
				logger.Println("Err: connection closed")
				time.Sleep(time.Second*2)
				break
			}
			logger.Printf("Received: %q: %q", msg.Topic, msg.Data)
			queue <- msg
		}
	}
}


////////////////////////////////////////////////////////////////////////////////

var ErrNoID = errors.New("no deviceID found")

func PostData(src int, data []byte) (err error) {

	values, err := parseCPham(data)
	if err != nil {
		return err
	}
	rawID := values["uid"]
	if rawID == nil {
		rawID = values["UID"]
	}
	var deviceID string
	if rawID != nil {
		var ok bool
		deviceID, ok = rawID.(string)
		if !ok {
			return ErrInvalidID
		}
		delete(values, "uid")
		delete(values, "UID")
	}
	if deviceID == "" {
		deviceID, _ = GetLocalID()
		deviceID += "_"+strconv.Itoa(src)
	}

	for key, value := range values {
		err = CreateValue(deviceID, key, value)
	}
	return
}

func CreateValue(deviceID string, sensorID string, value interface{}) error {
	data, _ := json.Marshal(value)
	url := EdgeOrigin+"/devices/"+deviceID+"/sensors/"+sensorID+"/value"
	resp, err := http.Post(url, ContentType, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		err = CreateSensor(deviceID, sensorID)
		if err != nil {
			return err
		}
		resp, err = http.Post(url, ContentType, bytes.NewBuffer(data))
		if err != nil {
			return err
		}
	}
	if !isOk(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		logger.Printf("Err: %s: %q %s", resp.Status, url, body)
		return fmt.Errorf("http response %d", resp.StatusCode)
	}
	return nil
}

func CreateSensor(deviceID string, sensorID string) error {
	var sensor = struct{
		ID string `json:"id"`
		Name string `json:"name"`
	}{sensorID, sensorID};
	data, _ := json.Marshal(sensor)
	url := EdgeOrigin+"/devices/"+deviceID+"/sensors"
	resp, err := http.Post(url, ContentType, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		err = CreateDevice(deviceID)
		if err != nil {
			return err
		}
		resp, err = http.Post(url, ContentType, bytes.NewBuffer(data))
		if err != nil {
			return err
		}
	}
	if !isOk(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		logger.Printf("Err: %s: %q %s", resp.Status, url, body)
		return fmt.Errorf("http response %d", resp.StatusCode)
	} else {
		logger.Printf("Created sensor \"devices/%s/sensors/%s\".", deviceID, sensorID)
	}
	return nil
}

func CreateDevice(deviceID string) error {
	var device = struct{
		ID string `json:"id"`
		Name string `json:"name"`
	}{deviceID, "LoRa Device "+deviceID};
	data, _ := json.Marshal(device)
	url := EdgeOrigin+"/devices"
	resp, err := http.Post(url, ContentType, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if !isOk(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		logger.Printf("Err: %s: %q %s", resp.Status, url, body)
		return fmt.Errorf("http response %d", resp.StatusCode)
	} else {
		logger.Printf("Created device \"devices/%s\".", deviceID)
	}
	return nil
}

var ErrInvalidData = errors.New("malformed cpham payload")
var ErrInvalidID = errors.New("uid field must be string")

func parseCPham(data []byte) (fields map[string]interface{}, err error) {
	fields = make(map[string]interface{})
	for len(data) != 0 {
		i := 0
		for i != len(data) && data[i] != '/' {
			i++
		}
		key := string(data[:i])
		if i == 0 || i == len(data) {
			err = ErrInvalidData
			return
		}
		time.Sleep(1*time.Second)
		data = data[i+1:]
		i = 0
		open := 0
		inStr := false
		for i < len(data) {
			if !inStr && data[i] == '"' {
				inStr = true
				i++
				continue
			}
			if inStr {
				if data[i] == '\\' {
					i += 2
					continue
				}
				if data[i] == '"' {
					inStr = false
					i++
				}
				continue
			}
			if data[i] == '{' {
				open++
				i++
				continue
			}
			if data[i] == '}' {
				open--
				i++
				continue
			}
			if open == 0 && data[i] == '/' {
				break
			}
			i++
		}
		if i > len(data) {
			err = ErrInvalidData
			return
		}
		var value interface{}
		err = json.Unmarshal(data[:i], &value)
		if err != nil {
			value = string(data[:i])
		}
		time.Sleep(1*time.Second)
		if i < len(data) {
			data = data[i+1:]
		} else {
			data = nil
		}
		fields[key] = value
	}
	return
}

var localID string

func GetLocalID() (string, error) {
	if localID != "" {
		return localID, nil
	}
	resp, err := http.Get(EdgeOrigin+"/device/id")
	if err != nil {
		logger.Printf("Err: %v", err)
		return "", err
	}
	if !isOk(resp.StatusCode) {
		return "", fmt.Errorf("http response %d", resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("Err: %v", err)
		return  "", fmt.Errorf("http response %d: can not read body", resp.StatusCode)
	}
	localID = string(data)
	return localID, nil
}

func isOk(status int) bool {
	return status >= 200 && status < 300
}