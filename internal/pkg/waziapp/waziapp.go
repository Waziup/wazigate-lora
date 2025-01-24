package waziapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
)

var Dir = "/var/lib/waziapp"

var Name string

var Version string

type PackageJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

func ProvidePackageJSON(file []byte) error {
	var packageJSON PackageJSON
	if err := json.Unmarshal(file, &packageJSON); err != nil {
		return err
	}
	Name = packageJSON.Name
	Version = packageJSON.Version
	if err := os.WriteFile(path.Join(Dir, "package.json"), file, 0777); err != nil {
		logHints()
		return err
	}
	return nil
}

func ProvideListener() (l net.Listener, err error) {
	proxySock := path.Join(Dir, "proxy.sock")
	if err := os.Remove(proxySock); err != nil && !os.IsNotExist(err) {
		log.Printf("This WaziApp's 'proxy.sock' file in '%s' could not be deleted.", Dir)
		logHints()
		return nil, err
	}
	l, err = net.Listen("unix", proxySock)
	if err != nil {
		log.Printf("This WaziApp's 'proxy.sock' file in '%s' could not be created.", Dir)
		logHints()
		return nil, err
	}
	return l, err
}

func ListenAndServe(handler http.Handler) error {
	proxySock := path.Join(Dir, "proxy.sock")
	listener, err := ProvideListener()
	if err != nil {
		return err
	}
	defer listener.Close()
	defer os.Remove(proxySock)
	return http.Serve(listener, handler)
}

func logHints() {
	log.Printf("Make sure to run this container with the mapped volume '%s'.", Dir)
	log.Println("See the WaziApp documentation for more details on running WaziApps.")
}

const HealthcheckPath = "/healtcheck"

func Healtcheck() error {
	proxySock := path.Join(Dir, "proxy.sock")
	conn, err := net.Dial("unix", proxySock)
	if err != nil {
		return err
	}
	// the connection will be closed when the function returns
	defer conn.Close()

	// building a http transport that uses the unix socket
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			// the proxy uses linux sockets that are created by each app
			return conn, nil
		},
		MaxConnsPerHost:   1,
		MaxIdleConns:      1,
		DisableKeepAlives: true,
	}
	client := http.Client{
		Transport: transport,
	}
	defer client.CloseIdleConnections()

	//

	healtcheckUrl := "http://localhost" + HealthcheckPath
	req, err := http.NewRequest(http.MethodGet, healtcheckUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "WaziApp")
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil // ok
}
