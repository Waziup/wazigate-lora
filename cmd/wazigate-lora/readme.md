## This is a Wazigate App.

# Wazigate LoRa

# Development

You need to build the wazigate-lora container and run it together with the 

```sh
docker run --rm \
	--net wazigate \
	--name wazigate-lora \
	-v "$PWD":/wazigate-lora \
	-w /wazigate-lora \
	golang:1.13-alpine \
	go run .
```