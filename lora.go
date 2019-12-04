package main

type Radio interface {
	On() error
	Off() error
	Send(payload []byte) error
}