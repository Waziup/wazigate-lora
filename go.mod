module github.com/Waziup/wazigate-lora

// replace github.com/Waziup/wazigate-rpi/gpio => ../wazigate-rpi/gpio

// replace github.com/Waziup/wazigate-rpi/spi => ../wazigate-rpi/spi

// replace github.com/Waziup/wazigate-edge/mqtt => ../wazigate-edge/mqtt

require (
	github.com/Waziup/wazigate-edge/mqtt v0.0.0-20191213091021-e016fed2ef89
	github.com/Waziup/wazigate-rpi/gpio v0.0.0-20191204155719-329ba73795d3
	github.com/Waziup/wazigate-rpi/spi v0.0.0-20191204155719-329ba73795d3
	golang.org/x/sys v0.0.0-20191110163157-d32e6e3b99c4
)

go 1.13
