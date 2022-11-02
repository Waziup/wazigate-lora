package waziapp

import (
	"encoding/json"
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
