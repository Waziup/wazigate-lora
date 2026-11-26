#!/bin/sh
# WaziGate targets arm64 (Pi 3/4/CM4 with the 64-bit kernel). This was
# linux/arm/v7, which no longer matches anything the project ships.
docker buildx build --platform=linux/arm64 --tag waziup/wazigate-lora --load
