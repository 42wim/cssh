package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/42wim/cssh/device"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var (
	host     string
	cmd      string
	cmdfrom  string
	hostfrom string
	cfg      config
	logdir   string
	pipe     bool
)

func init() {
	flag.StringVar(&host, "host", "", "host")
	flag.StringVar(&cmd, "cmd", "", "single command to execute")
	flag.StringVar(&cmdfrom, "cmd-from", "", "file containing commands to execute")
	flag.StringVar(&hostfrom, "host-from", "", "file containing hosts")
	flag.StringVar(&logdir, "logdir", "", "directory to log output")
	flag.BoolVar(&pipe, "pipe", false, "pipe to stdin")
	flag.Parse()
	cfg = readConfig("cssh.conf")
}

func doCmd(d *device.CiscoDevice) {
	d.Cmd("terminal length 0")
	output, err := d.Cmd(cmd)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(output)
	fmt.Println()
}

func doCmdFrom(d *device.CiscoDevice) {
	dat, _ := ioutil.ReadFile(cmdfrom)
	lines := strings.Split(string(dat), "\n")
	d.Cmd("terminal length 0")
	for _, line := range lines {
		if line != "" {
			if strings.Contains(line, "#") {
			} else {
				output, err := d.Cmd(line)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Print(output)
			}
		}
	}
}

func doHost(host string, c chan int) {
	if host != "" {
		cred := getCredentials(host)
		if cred.Username == "" {
			fmt.Println("couldn't get credentials for " + host)
			c <- 2
			return
		}
		d := &device.CiscoDevice{Hostname: host, Username: cred.Username, Password: cred.Password, Enable: cred.Enable, Logdir: logdir}
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

func doPipe(host string) {
	cred := getCredentials(host)
	if cred.Username == "" {
		fmt.Println("couldn't get credentials for " + host)
		return
	}
	d := &device.CiscoDevice{Hostname: host, Username: cred.Username, Password: cred.Password, Enable: cred.Enable, Logdir: logdir}
	err := d.Connect()
	if err != nil {
		log.Println("couldn't connect", err.Error())
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	if err != nil {
		log.Println(err)
		return
	}
	d.Echo = false
	d.EnableLog = false
	d.Cmd("terminal length 0")
	d.EnableLog = true
	output, _ := d.Cmd("")
	fmt.Print(output)
	if d == nil {
		fmt.Println("something went wrong in doPipe(" + host + ")")
		return
	}
	for scanner.Scan() {
		cmd := scanner.Text()
		output, _ := d.Cmd(cmd)
		fmt.Print(output)
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
	if pipe && host != "" {
		doPipe(host)
		return
	}

	if (cmd != "" || cmdfrom != "") && host != "" {
		c := make(chan int, 1)
		go doHost(host, c)
		<-c
	}
}
