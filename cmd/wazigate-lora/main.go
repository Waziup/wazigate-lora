// This service creates a bridge between Chirpstack and Wazigate.
// Every Wazigate Device might implement 'lorawan' metadata that will make this service create a
// a identical device in Chirpstack. The metadata should look like this:
//
//	{
//	  "lorawan": {
//	     "devEUI": "AA555A0026011d87",
//	     "profile": "WaziDev",
//	     "devAddr": "26011d87",
//	     "appSKey": "23158d3bbc31e6af670d195b5aed5525",
//	     "nwkSEncKey": "d83cb057cebd2c43e21f4cde01c19ae1",
//	   }
//	}
package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Waziup/wazigate-lora/internal/app"
	"github.com/Waziup/wazigate-lora/internal/pkg/waziapp"
	"github.com/Waziup/wazigate-lora/internal/pkg/wazigate"
)

//go:embed package.json
var packageJSON []byte

func main() {
	log.SetFlags(0)
	////////////////////

	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := waziapp.Healtcheck(); err != nil {
			log.Fatalf("Healthcheck failed: %v", err)
		}
		return
	}

	////////////////////

	if err := waziapp.ProvidePackageJSON(packageJSON); err != nil {
		log.Fatalf("Can not provide package.json: %v", err)
	}
	log.Printf("Starting...")
	log.Printf("This is the WaziGate-LoRa App (%s) v%s.", waziapp.Name, waziapp.Version)

	if err := app.ReadConfig(); err != nil {
		log.Fatalf("Can not read config: %v", err)
	}

	if err := wazigate.Connect(); err != nil {
		log.Fatalf("Can not connect to WaziGate: %v", err)
	}
	////////////////////

	for {
		// read the gateway ID and store it in the chirpstack.json file
		id, err := wazigate.ID()
		if err == nil {
			log.Printf("Gateway ID: %s", id)
			// Chirpstack requires a 8-Byte Gateway Id, but the Wazigate Id might have a different length (usually a 6-Byte MAC addr).
			// By commenting the following line, the default Id from chirpsstack.json remains unchanged.
			// config.Gateway.Id = id
			app.Config.Gateway.Name = "LocalWazigate_" + id // We need it apparently CS fails to create a GW if there is already one with the same name
			if err := app.WriteConfig(); err != nil {
				panic(fmt.Errorf("can not write 'chirpstack.json': %v", err))
			}
			break
		}
		log.Printf("Err Can not get Wazigate ID: %v", err)
		log.Println("Can not call edge API, waiting for some seconds ...")
		time.Sleep(3 * time.Second)
	}

	////////////////////

	go app.ListenAndServe()

	for {
		err := app.InitChirpstack()
		if err == nil {
			break
		}
		log.Printf("Err %v", err)
		time.Sleep(time.Second * 5)
	}

	for {
		app.InitDevice()
		err := app.Serve()
		log.Printf("Err %v", err)
		time.Sleep(time.Second * 5)
	}
}
