package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-lora/app/internal/pkg/fetch"
)

// Device represents a Waziup Device.
type Device struct {
	Name      string      `json:"name" bson:"name"`
	ID        string      `json:"id" bson:"_id"`
	Sensors   []*Sensor   `json:"sensors" bson:"sensors"`
	Actuators []*Actuator `json:"actuators" bson:"actuators"`
	Modified  time.Time   `json:"modified" bson:"modified"`
	Created   time.Time   `json:"created" bson:"created"`
	Meta      Meta        `json:"meta" bson:"meta"`
}

// DevicesQuery is used to range or limit query results.
type DevicesQuery struct {
	Limit int64
	Size  int64
	Meta  []string
}

// Meta holds entity metadata.
type Meta map[string]interface{}

func (m Meta) Get(key string) JSON {
	return JSON{m[key]}
}

// Sensor represents a Waziup sensor
type Sensor struct {
	ID       string      `json:"id" bson:"id"`
	Name     string      `json:"name" bson:"name"`
	Modified time.Time   `json:"modified" bson:"modified"`
	Created  time.Time   `json:"created" bson:"created"`
	Time     *time.Time  `json:"time" bson:"time"`
	Value    interface{} `json:"value" bson:"value"`
	Meta     Meta        `json:"meta" bson:"meta"`
}

// Actuator represents a Waziup actuator
type Actuator struct {
	ID       string      `json:"id" bson:"id"`
	Name     string      `json:"name" bson:"name"`
	Modified time.Time   `json:"modified" bson:"modified"`
	Created  time.Time   `json:"created" bson:"created"`
	Time     *time.Time  `json:"time" bson:"time"`
	Value    interface{} `json:"value" bson:"value"`
	Meta     Meta        `json:"meta" bson:"meta"`
}

////////////////////////////////////////////////////////////////////////////////

// Waziup represents the API of the Waziup Cloud or a Wazigate.
type Waziup struct {
	Host       string
	Token      string
	MQTTClient *mqtt.Client
}

var defaultEdgeHost = "wazigate-edge"

func getEdgeHost() string {
	edgeHost := os.Getenv("WAZIGATE_EDGE")
	if edgeHost == "" {
		return defaultEdgeHost
	}
	return edgeHost
}

// Wazigate is the API of Wazigate.
var Wazigate, _ = Connect(&ConnectSettings{
	Host: getEdgeHost(),
})

// ConnectSettings are used when connecting to a Waziup API via `Connect`.
type ConnectSettings struct {
	Host     string
	Token    string
	Password string
	Username string
}

// Connect to a Waziup API.
func Connect(settings *ConnectSettings) (*Waziup, error) {
	var cloud = &Waziup{}
	if settings != nil {
		cloud.Host = settings.Host
		cloud.Token = settings.Token
	}
	return cloud, nil
}

var jsonContentTypeRegexp = regexp.MustCompile("^application/json(;|$)")

// Get queries an API endpoint to read data.
func (w *Waziup) Get(res string, o interface{}) error {
	resp := fetch.Fetch(w.ToURL(res), nil)
	log.Printf("GET %d %s %q", resp.Status, res, resp.StatusText)
	if !resp.OK {
		text, _ := resp.Text()
		log.Println(text)
		return fmt.Errorf("fetch: %s", resp.StatusText)
	}
	contentType := resp.Headers.Get("Content-Type")
	if o == nil {
		return resp.Body.Close()
	}
	if jsonContentTypeRegexp.MatchString(contentType) {
		return resp.JSON(o)
	}
	str, ok := o.(*string)
	if !ok {
		return fmt.Errorf("fetch: %s: Can not unmarshal %q", res, contentType)
	}
	var err error
	*str, err = resp.Text()
	return err
}

type Error struct {
	URL        string
	Status     int
	StatusText string
	Text       string
}

func (err *Error) Error() string {
	return fmt.Sprintf("fetch: %d %s %q\n%s", err.Status, err.StatusText, err.URL, err.Text)
}

func IsNotExist(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Status == 404
	}
	return false
}

// Set queries an API endpoint to write data.
func (w *Waziup) Set(res string, i interface{}, o interface{}) error {
	body, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("fetch: can not marshal request: %v", err)
	}
	resp := fetch.Fetch(w.ToURL(res), &fetch.FetchInit{
		Method: "POST",
		Body:   bytes.NewBuffer(body),
		Headers: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
	})
	log.Printf("POST %s: %d %s", res, resp.Status, resp.StatusText)
	if !resp.OK {
		text, _ := resp.Text()
		return &Error{
			URL:        res,
			Status:     resp.Status,
			StatusText: resp.StatusText,
			Text:       text,
		}
	}
	contentType := resp.Headers.Get("Content-Type")
	if o == nil {
		return resp.Body.Close()
	}
	if jsonContentTypeRegexp.MatchString(contentType) {
		return resp.JSON(o)
	}
	str, ok := o.(*string)
	if !ok {
		return fmt.Errorf("fetch: %s: Can not unmarshal %q", res, contentType)
	}
	*str, err = resp.Text()
	return err
}

// ToURL returns the absolute URL of an API endpoint path.
func (w *Waziup) ToURL(path string) string {
	// return "http://" + w.Host + ":880/" + path
	return "http://" + w.Host + "/" + path
}

// GetID returns the Wazigate ID.
func (w *Waziup) GetID() (id string, err error) {
	err = w.Get("device/id", &id)
	return
}

// AddSensorValue uploads a new sensor value to the Waziup Cloud or Gateway.
func (w *Waziup) AddSensorValue(deviceID string, sensorID string, value interface{}) error {
	return w.Set("devices/"+deviceID+"/sensors/"+sensorID+"/value", value, nil)
}

func (w *Waziup) UnmarshalDevice(deviceID string, data []byte) error {
	resp := fetch.Fetch(w.ToURL("devices/"+deviceID), &fetch.FetchInit{
		Method: "POST",
		Body:   bytes.NewBuffer(data),
		Headers: http.Header{
			"Content-Type": []string{"application/octet-stream"},
		},
	})
	text, _ := resp.Text()
	if !resp.OK {
		return fmt.Errorf("UnmarshalDevice: Err %d: %s %s", resp.Status, resp.StatusText, text)
	}
	if text != "" {
		log.Printf("UnmarshalDevice: Server says %q", text)
	}
	return nil
}

func (w *Waziup) MarshalDevice(deviceID string) ([]byte, error) {
	resp := fetch.Fetch(w.ToURL("devices/"+deviceID), &fetch.FetchInit{
		Method: "GET",
		Headers: http.Header{
			"Content-Type": []string{"application/octet-stream"},
		},
	})
	data, _ := resp.Bytes()
	if !resp.OK {
		return data, fmt.Errorf("MarshalDevice: Err %d: %s %s", resp.Status, resp.StatusText, string(data))
	}
	return data, nil
}

func (w *Waziup) AddSensor(deviceID string, sensor *Sensor) error {
	return w.Set("devices/"+deviceID+"/sensors", sensor, &sensor.ID)
}

// GetDevices queries all devices.
func (w *Waziup) GetDevices(query *DevicesQuery) (devices []Device, err error) {
	res := "devices"
	if query != nil {
		q := make(map[string]string)
		if query.Limit != 0 {
			q["limit"] = strconv.FormatInt(query.Limit, 10)
		}
		if query.Limit != 0 {
			q["size"] = strconv.FormatInt(query.Size, 10)
		}
		if query.Meta != nil {
			q["meta"] = strings.Join(query.Meta, ",")
		}
		res += toQuery(q)
	}
	err = w.Get(res, &devices)
	return
}

func toQuery(query map[string]string) (rawQuery string) {
	first := true
	for k, v := range query {
		if first {
			rawQuery = "?"
		} else {
			rawQuery += "&"
		}
		first = false
		if v == "" {
			rawQuery += k
		} else {
			rawQuery += k + "=" + v
		}
	}
	return
}

////////////////////////////////////////////////////////////////////////////////

func (w *Waziup) ConnectMQTT() (err error) {
	if w.MQTTClient == nil {
		id := "wazigate-lora-" + strconv.Itoa(rand.Int())
		w.MQTTClient, err = mqtt.Dial(w.Host+":1883", id, true, nil, nil)
	}
	return
}

func (w *Waziup) DisconnectMQTT() (err error) {
	if w.MQTTClient != nil {
		err = w.MQTTClient.Disconnect()
		w.MQTTClient = nil
	}
	return
}

func (w *Waziup) Subscribe(topic string) (err error) {
	if err := w.ConnectMQTT(); err != nil {
		return err
	}
	_, err = w.MQTTClient.Subscribe(topic, 0)
	if err != nil {
		w.DisconnectMQTT()
	}
	return err
}

func (w *Waziup) Message() (*mqtt.Message, error) {
	if err := w.ConnectMQTT(); err != nil {
		return nil, err
	}
	msg, err := w.MQTTClient.Message()
	if err != nil {
		w.DisconnectMQTT()
	}
	return msg, err
}

////////////////////////////////////////////////////////////////////////////////

var errNoValue = errors.New("no value")

type JSON struct {
	value interface{}
}

func (json JSON) Get(key string) JSON {
	if json.value == nil {
		return JSON{}
	}
	m, ok := json.value.(map[string]interface{})
	if !ok {
		return JSON{}
	}
	return JSON{value: m[key]}
}

func (json JSON) Undefined() bool {
	return json.value == nil
}

func (json JSON) String() (string, error) {
	if json.value == nil {
		return "", errNoValue
	}
	s, ok := json.value.(string)
	if !ok {
		return "", errNoValue
	}
	return s, nil
}

func (json JSON) Number() (float64, error) {
	if json.value == nil {
		return 0, errNoValue
	}
	n, ok := json.value.(float64)
	if !ok {
		return 0, errNoValue
	}
	return n, nil
}

func (json JSON) Int() (int, error) {
	if json.value == nil {
		return 0, errNoValue
	}
	n, ok := json.value.(float64)
	if !ok {
		return 0, errNoValue
	}
	return int(n), nil
}
