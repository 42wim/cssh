package main

import (
	"flag"
	"fmt"
	"github.com/42wim/cssh/device"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

var host string
var cmd string
var cmdfrom string
var hostfrom string
var cfg config

func init() {
	flag.StringVar(&host, "host", "", "host")
	flag.StringVar(&cmd, "cmd", "", "single command to execute")
	flag.StringVar(&cmdfrom, "cmd-from", "", "file containing commands to execute")
	flag.StringVar(&hostfrom, "host-from", "", "file containing hosts")
	flag.Parse()
	cfg = readConfig("cssh.conf")
}

func doCmd(d *device.CiscoDevice) {
	d.Cmd("terminal length 0")
	fmt.Print(d.Cmd(cmd))
	fmt.Println()
}

func doCmdFrom(d *device.CiscoDevice) {
	dat, _ := ioutil.ReadFile(cmdfrom)
	lines := strings.Split(string(dat), "\n")
	d.Cmd("terminal length 0")
	for _, line := range lines {
		if line != "" {
			fmt.Print(d.Cmd(line))
		}
	}
	fmt.Println()
}

func doHost(host string, c chan int) {
	if host != "" {
		//fmt.Println("connecting to " + host)
		cred := getCredentials(host)
		if cred.Username == "" {
			fmt.Println("couldn't get credentials for " + host)
			c <- 2
			return
		}
		d := &device.CiscoDevice{Hostname: host, Username: cred.Username, Password: cred.Password, Enable: cred.Enable}
		err := d.Connect()
		if err != nil {
			log.Println("couldn't connect", err.Error())
			c <- 1
			return
		} else {
			defer d.Close()
			if cmdfrom != "" {
				doCmdFrom(d)
			}
			if cmd != "" {
				doCmd(d)
			}
			c <- 1
		}
	}
}

func main() {
	if host == "" && hostfrom == "" {
		flag.PrintDefaults()
		return
	}
	if hostfrom != "" {
		dat, _ := ioutil.ReadFile(hostfrom)
		lines := strings.Split(string(dat), "\n")
		// make buffers
		c := make(chan int, len(lines)-1)
		for _, host := range lines[0 : len(lines)-1] {
			time.Sleep(time.Millisecond * 50)
			go doHost(host, c)
		}
		// wait for all goroutines
		for i := 0; i < len(lines)-1; i++ {
			<-c
		}
		return
	}
	if (cmd != "" || cmdfrom != "") && host != "" {
		c := make(chan int, 1)
		go doHost(host, c)
		<-c
	}
}
