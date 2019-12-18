package main

import (
	"encoding/json"
	"errors"
	"time"
)

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
		time.Sleep(1 * time.Second)
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
		time.Sleep(1 * time.Second)
		if i < len(data) {
			data = data[i+1:]
		} else {
			data = nil
		}
		fields[key] = value
	}
	return
}
