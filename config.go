package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Username                 string
	BufferAddSep             string
	ClearBufferOnSend        bool
	DefaultHost, DefaultPort string
}

func getConfig() (conf Config, confPath string, err error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	if runtime.GOOS == "darwin" {
		var home string
		home, err = os.UserHomeDir()
		if err != nil {
			return
		}

		configDir = filepath.Join(home, ".config")
	}

	dirPath := filepath.Join(configDir, "gnc")
	err = os.MkdirAll(dirPath, os.ModeDir|0o777)
	if err != nil {
		return
	}

	confPath = filepath.Join(dirPath, "config.toml")

	f, err := os.Open(confPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			f, err = os.Create(confPath)
			if err != nil {
				return
			}

			defer f.Close()

			conf.Username = ""
			conf.BufferAddSep = " "
			conf.ClearBufferOnSend = true
			conf.DefaultHost = "localhost"
			conf.DefaultPort = "44322"

			enc := toml.NewEncoder(f)
			err = enc.Encode(&conf)
			return
		} else {
			err = fmt.Errorf("Cannot open config file '%s': %s", confPath, err.Error())
			return
		}
	}

	defer f.Close()

	dec := toml.NewDecoder(f)
	_, err = dec.Decode(&conf)
	if err != nil {
		return
	}

	conf.Username = strings.TrimSpace(conf.Username)

	return
}
