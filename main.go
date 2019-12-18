package main

import (
	"log"
	"os"
	"strings"
	"time"
	"net/http"

	"github.com/Waziup/wazigate-edge/mqtt"
)

const EdgeOrigin = "http://127.0.0.1:880"
const ContentType = "application/json; charset=utf-8"
const Radios = "sx127x,sx1272,sx1276,sx1301"

var queue = make(chan *mqtt.Message, 8)
var offline = false

var logger *log.Logger

func main() {
	log.SetFlags(0)
	logger = log.New(os.Stdout, "[     ] ", 0)

	listenAddr := os.Getenv("LISTEN_ADDR")

	radios := Radios
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-h", "h", "-help", "help":
			log.Println("Usage: wazigate-lora -r {radios} -o")
			log.Println("Arguments:")
			log.Println("  -r {radios}: Set radio chips.")
			log.Println("     List of comma separated radios: one or more of sx127x, sx1301")
			log.Println("  -o: Offline usage.")
			log.Println("     Do not forward packages.")
			log.Println("  -l: HTTP listen address.")
			log.Println("     The HTTP server will serve on that port.")
			log.Println("     Also use env LISTEN_ADDR.")
			log.Println("  -h: Show help (this screen).")
			log.Println("Examples:")
			log.Println("  wazigate-lora -r sx127x")
			log.Println("  wazigate-lora -r \"sx1301,sx127x\"   <-- default")
			log.Println("wazigate-lora")
			os.Exit(0)
		case "-r", "-radios":
			i++
			if i == len(os.Args) {
				logger.Fatalf("Err: argument %q is missing its value", arg)
			}
			radios = strings.ToLower(os.Args[i])
			if !strings.Contains(radios, "sx127x") && !strings.Contains(radios, "sx1301") {
				logger.Fatalf("none of the radios match %q.", Radios)
			}
		case "-l", "-listen":
			i++
			if i == len(os.Args) {
				logger.Fatalf("Err: argument %q is missing its value", arg)
			}
			listenAddr = os.Args[i]

		case "-o", "-offline":
			offline = true
		default:
			logger.Fatalf("Err: unrecognized argument: %q", arg)
		}
	}

	if !offline {
		id, err := GetLocalID()
		if err != nil {
			logger.Printf("Err: can not connect to local edge service: %v", err)
		} else {
			logger.Printf("Edge ID: %q", id)
		}
		id = ""
		err = nil
		go downstream()
	}

	if listenAddr != "" {
		logger.Printf("Listening on %q, serving from \"www\".", listenAddr)
		go func(addr string) {
			log.Fatal(http.ListenAndServe(addr, http.FileServer(http.Dir("www"))))
		}(listenAddr)
	}

	changed, err := fetchConfig()
	if err != nil {
		logger.Printf("Err: can not read device config: %v", err)
	} else if !changed {
		printConfig()
	}

	logger.Println("Looking for connected radios...")

	for true {
		if strings.Contains(radios, "sx127x") {
			if sx127x() {
				time.Sleep(time.Second / 2)
				continue
			}
			time.Sleep(time.Second / 2)
		}
		if strings.Contains(radios, "sx1301") {
			if sx1301() {
				time.Sleep(time.Second / 2)
				continue
			}
			time.Sleep(time.Second / 2)
		}
	}
}