package waziapp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/Waziup/wazigate-lora/internal/pkg/wazigate"
)

var ConfigDir string

const ConfigFile = "config.json"

func ReadConfig(config interface{}) (err error) {
	var configDirs = []string{
		// for development, search for the config.json in the current directory first
		".",
		// for WaziApps, all persistent files go to /var/lib/waziapp
		Dir,
	}

	if Name != "" {
		configDirs = append(configDirs,
			// for development on the Wazigate host system, search in the WaziGate apps directory
			wazigate.AppDir(Name), // /var/lib/wazigate/waziup.wazigate-lora
			// for development, search in the /etc/ dir
			"/etc/"+Name,
		)
	}

	var file []byte
	for _, dir := range configDirs {
		file, err = ioutil.ReadFile(filepath.Join(dir, ConfigFile))
		if err != nil {
			err = fmt.Errorf("can not open '%s': %v", ConfigFile, err)
			continue
		}
		if err = json.Unmarshal(file, config); err != nil {
			return fmt.Errorf("can not parse '%s': %v", ConfigFile, err)
		}
		ConfigDir = dir
		return nil
	}
	return
}

func WriteConfig(config interface{}) error {
	file, _ := json.MarshalIndent(config, "", "  ")
	if err := ioutil.WriteFile(filepath.Join(ConfigDir, ConfigFile), file, 0666); err != nil {
		return fmt.Errorf("can not write '%s': %v", ConfigFile, err)
	}
	return nil
}
