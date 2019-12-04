module github.com/Waziup/wazigate-lora

replace github.com/Waziup/wazigate-rpi/gpio => ../wazigate-rpi/gpio

replace github.com/Waziup/wazigate-rpi/spi => ../wazigate-rpi/spi

replace github.com/Waziup/wazigate-edge/mqtt => ../wazigate-edge/mqtt

require (
	github.com/Waziup/wazigate-edge/mqtt v1.0.0
	github.com/Waziup/wazigate-rpi/gpio v1.0.0
	github.com/Waziup/wazigate-rpi/spi v1.0.0
	golang.org/x/sys v0.0.0-20191110163157-d32e6e3b99c4
)

go 1.13
