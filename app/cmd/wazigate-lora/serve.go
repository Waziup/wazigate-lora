package main

import (
	"log"
	"net/http"
)

func serveHTTP(resp http.ResponseWriter, req *http.Request) {

	var w = wrapper{
		ResponseWriter: resp,
		statusCode:     200,
	}
	serveAPI(&w, req)
	log.Printf("[%s] %d, %s %s s:%d", req.RemoteAddr, w.statusCode, req.Method, req.RequestURI, w.size)
}

type wrapper struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *wrapper) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func (w *wrapper) Write(d []byte) (s int, err error) {
	s, err = w.ResponseWriter.Write(d)
	w.size += s
	return
}

var server = http.FileServer(http.Dir("www"))

func serveStatic(resp http.ResponseWriter, req *http.Request) {
	server.ServeHTTP(resp, req)
}
