# WaziGate LoRa

WaziGate LoRa ```wazigate-lora``` is a WaziApp that enables the [WaziGate](https://github.com/Waziup/WaziGate) to communicate with LoRaWAN® devices. It does so by connection the [ChirpStack open-source LoRaWAN® Network Serve](https://www.chirpstack.io/) to the WaziGate via API. WaziGate LoRa acts as a data broker and does not store any data nor does it provide a user interface. It is a service that runs in the background and forwards data from the LoRaWAN® network to the WaziGate and vice versa.

The features of WaziGate LoRa are:

- creating a ChirpStack LoRaWAN® device for each LoRaWAN-enabled WaziGate device,
- forwarding data from the LoRaWAN® network to the WaziGate (Uplinks),
- forwarding data from the WaziGate to the LoRaWAN® network (Downlinks),

WaziGate LoRa commes pre-installed on the WaziGate.

# Data Flow

WaziGate LoRa connects to the Gateway's MQTT instance to listen for Wazigate Edge data and
Chirpstack messages. It then forwards the data to the other service by calling the respective API.

The software maintains an internal mapping of the WaziGate devices to the ChirpStack devices, by keeping track of WaziGate device IDs and device metadata and ChirpStack device EUIs (DevEUI).

WaziGate device metadata is expected to contain the following fields:

```json
{
  "lorawan": {
    "devEUI": "AA555A0026011DD1",
    "devAddr": "26011DD1",
    "appSKey": "23158D3BBC31E6AF670D195B5AED5525",
    "nwkSEncKey": "23158D3BBC31E6AF670D195B5AED5525",
    "profile": "WaziDev"
  }
}
```

The `devEUI` field is the unique identifier of the device in the LoRaWAN® network. The `devAddr`, `appSKey`, and `nwkSEncKey` are the LoRaWAN® keys used for encryption and decryption of the data if using Activation By Personalization (ABP) method. The `profile` field is the name of ChirpStack device profile that should be used for this device.

It listens to the following MQTT topics:

- `eu868/gateway/+/event/+` for ChirpStack gateway events

  The `up` (Uplink) messages contains data about the received LoRa frames, even if the sender is not handled by our network.
  We use this information for logging purposes.

- `application/+/device/+/event/+` for ChirpStack application device events

  The `up` (Uplink) messages contains data about the received LoRaWAN® messages and the decrypted payload for devices registered with ChirpStack. The binary payload is not parsed but forwarded as is to the WaziGate by posting to the `/devices/{id}` endpoint, triggering the WaziGate codec to parse the data, possibly creating sensors and measurements.

  Some additional events like `ack` (Acknowledgement), `status` (Device status), `txack` (Downlink acknowledgement), and `error` (Error) are collected for logging purposes.

- `devices/+/actuators/+/value[s]` for WaziGate actuator commands

  This topic is triggerd when a WaiGate devices receives a new actuator value. Instead of just forwarding the value, we call the device's codec to encode all the actuators of that device and send the encoded payload to the ChirpStack network enqueueing a downlink message.

- `devices/+/meta` for WaziGate device meta

  Connection between WaziGate devices and ChirpStack devices is maintained by keeping track of the `lorawan` field in the WaziGate device metadata. This topic is used to update the metadata to collect updates of said field. We keep record of the LoRaWAN `devEUI` and the WaziGate device ID to link ChirpStack and WaziGate devices. Adding the `lorawan` field to the device metadata will create a new ChirpStack device and link it to the WaziGate device.

- `devices` for WaziGate device creation

  When a new device is created on the WaziGate, we will create a new ChirpStack device if the `lorawan` field is present in the device metadata. Deletion of the device is not mirrored to ChirpStack for now. A deleted Wazigate device will still be present in ChirpStack and might continue to receive uplinks from the network.

When starting for the first time, the service will setup ChirpStack by creating necessary devices profiles and applications. it will also create a ChirpStack device for each WaziGate device that has the `lorawan` field in its metadata.

WaziGate LoRa does not feature a user interface and does not provide an API. Relational data is stored in memory and is not persisted. The service is started as a background service and runs as a Docker container.

# Build and Deploy

This service is build as a docker container and runs on WaziGate as WaziApp. It comes pre-installed on the WaziGate. If you want to build it from source, you can do so by following these steps:

For compiling the docker container to be used on the Raspberry Pi architecture, you need to cross-compiling the docker image like this:

```bash	
docker buildx build --platform linux/arm64 --tag waziup/wazigate-lora:latest --output type=docker,dest=- . > wazigate-lora-latest.tar
```

The above command will create a docker image for the ARM64 architecture and save it as a tar file. 

If you wish to replace the existing image on the WaziGate, you can do so by loading the tar file into the WaziGate and recreating the wazigate-lora service.

```bash
# load the image to docker
docker load -i wazigate-lora-latest.tar

# recreate the wazigate-lora service
cd /var/lib/wazigate
source .env
docker tag waziup/wazigate-lora:latest waziup/wazigate-lora:$WAZIGATE_TAG
docker-compose up wazigate-lora
```
