package main

import (
	"code.google.com/p/gcfg"
	"io/ioutil"
	"log"
	"regexp"
)

type Credentials struct {
	Username string
	Password string
	Enable   string
}

type config struct {
	Device map[string]*struct {
		Username string
		Password string
		Enable   string
	}
}

func readConfig(cfgfile string) config {
	var cfg config
	content, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		log.Fatal(err)
	}
	err = gcfg.ReadStringInto(&cfg, string(content))
	if err != nil {
		log.Fatal("Failed to parse "+cfgfile+":", err)
	}
	return cfg
}

func getCredentials(host string) Credentials {
	var cred Credentials
	for device, _ := range cfg.Device {
		re := regexp.MustCompile(device)
		if re.MatchString(host) {
			cred.Username = cfg.Device[device].Username
			cred.Password = cfg.Device[device].Password
			cred.Enable = cfg.Device[device].Enable
			break
		}
	}
	return cred
}
