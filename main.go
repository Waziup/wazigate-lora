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
    "github.com/Waziup/wazigate-rpi/gpio"
	"github.com/Waziup/wazigate-rpi/spi"
	"github.com/Waziup/wazigate-edge/mqtt"
)

const EdgeOrigin = "http://127.0.0.1:880"
const ContentType = "application/json; charset=utf-8"
const Radios = "sx127x,sx1272,sx1276,sx1301"

var radio Radio

var queue = make(chan *mqtt.Message, 8)

func sx1301() error {
	pinRst, err := gpio.Output(17)
	if err != nil {
        return err
	}
	sx := SX1301.New(pinRst)
	err = sx.On()
	if err != nil {
		return err
	}
	defer sx.Off()
	for true {
		pkts, err := sx.Receive()
		if err != nil {
			return err
		}
		logger.Println(pkts)
	}
	return nil
}

func sx127x() error {
	dev, err := spi.Open("/dev/spidev0.1", 1000000)
    dev.SetMode(0)
    dev.SetBitsPerWord(8)
    dev.SetLSBFirst(false)
	dev.SetMaxSpeed(1000000)
	defer dev.Close()

    if err != nil {
        return err
	}
	
	// SlaveSelect GPIO Pin
	pinSS, err := gpio.Output(8)
	if err != nil {
		return err
	}
	defer pinSS.Unexport()

	// Reset GIOP Pin
	pinRst, err := gpio.Output(4)
	if err != nil {
		return err
	}
	defer pinRst.Unexport()

	// SX127X instance
	sx := SX127X.New(dev, pinSS, pinRst) 

	// Log
	sx.Logger.SetOutput(os.Stdout)
	sx.Logger.SetPrefix("[LORA ] ")
	sx.LogLevel = SX127X.LogLevelNormal
	
	// Startup ...
	if err = sx.On(); err != nil {
		return err
	}
	if err = sx.SetMode(1); err != nil {
		return err
	}
	if err = sx.SetChannel(SX127X.CH_10_868); err != nil {
		return err
	}
	if err = sx.SetPowerDBM(14); err != nil {
		return err
	}

	var counter byte = 0

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

				if err := sx.Send(data); err != nil {
					logger.Printf("Err: %v", err)
				} else {
					log.Printf("[<<   ] %v", data)
				}
				counter++
			}
		}

		data, err := sx.ReceiveAll(SX127X.MAX_TIMEOUT)
		switch err {
		case nil:
			
			rssi, _ := sx.GetRSSIpacket();
			log.Printf("[>>   ] %v (rssi: %d)", data, rssi)

			if (len(data) > 6 && data[4] == '\\' && data[5] == '!') {
				dest := data[0]
				typ := data[1]
				src := data[2]
				num := data[3]
				logger.Printf("Dest: %d, Typ: %d, Src: %d, Num: %d, Data: %q, RSSI: %d", dest, typ, src, num, data[6:], rssi)
				err = PostData(int(src), data[6:])
				if err != nil {
					logger.Printf("Err: can not save data: %v", err)
				}
			} else {
				// LoRaWAN:
				// PHYPayload: MHDR | MACPayload | MIC
				// MHDR: MType | RFU | Major
				logger.Printf("Err: unknown receive format")
			}
		case SX127X.ErrTimeout:
		case SX127X.ErrIncorrectCRC:
			logger.Printf("Err: faulty package: incorrect CRC")
		default:
			// unknown error
			return err
		}
	}
	return nil // unreachable
}

var logger *log.Logger

func main() {
	log.SetFlags(0)
	logger = log.New(os.Stdout, "[     ] ", 0)

	id, err := GetLocalID()
	if err != nil {
		logger.Printf("Err: can not connect to local edge service: %v", err)
	} else {
		logger.Printf("Edge ID: %q", id)
	}
	id = ""
	err = nil

	go downstream()

	radios := Radios
	if len(os.Args) > 1 {
		radios = strings.ToLower(os.Args[1])
		if !strings.Contains(radios, "sx127x") && !strings.Contains(radios, "sx1301") {
			logger.Fatalf("Invalid argument: Use %q.", Radios)
		}
	}

	for true {
		if strings.Contains(radios, "sx127x") {
			logger.Println("Looking for SX127X...")
			err := sx127x()
			if err != nil {
				logger.Printf("Err: SX127X: %v", err)
			}
			time.Sleep(time.Second/2)
		}
		if strings.Contains(radios, "sx1301") {
			logger.Println("Looking for SX1301...")
			sx1301()
			if err != nil {
				logger.Printf("Err: SX1301: %v", err)
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