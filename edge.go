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
	"net/http"
    "github.com/Waziup/wazigate-lora/lora"
	"github.com/Waziup/wazigate-edge/mqtt"
)

func downstream() {
	logger := log.New(os.Stdout, "[Edge ] ", 0)
	for true {
		client, err := mqtt.Dial("127.0.0.1:1883", "wazigate-lora", true, nil, nil)
		if err != nil {
			logger.Printf("Err: %v", err)
			time.Sleep(time.Second*2)
			continue
		}
		logger.Println("Connected to edge mqtt.")
		client.Subscribe("device/meta", 0)

		fetchConfig()

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
			if msg.Topic == "device/meta" {
				var meta Meta
				err = json.Unmarshal(msg.Data, &meta)
				if err != nil {
					logger.Println("Err: can not unmarshal json: %v", err)
				} else {
					if meta.LoraMeta != nil {
						applyConfig(*meta.LoraMeta)
					}
				}
				continue
			}
			queue <- msg
		}
	}
}


////////////////////////////////////////////////////////////////////////////////


func process(pkt *lora.RxPacket) {
	if (len(pkt.Data) > 6 && pkt.Data[4] == '\\' && pkt.Data[5] == '!') {
		dest := pkt.Data[0]
		typ := pkt.Data[1]
		src := pkt.Data[2]
		num := pkt.Data[3]
		logger.Printf("<Congduc> Dest: %d, Typ: %d, Src: %d, Num: %d, Data: %q, RSSI: %d", dest, typ, src, num, pkt.Data[6:], pkt.RSSI)
		err := PostData(int(src), pkt.Data[6:])
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


////////////////////////////////////////////////////////////////////////////////

var ErrNoID = errors.New("no deviceID found")

func PostData(src int, data []byte) (err error) {
	if offline {
		return nil
	}

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