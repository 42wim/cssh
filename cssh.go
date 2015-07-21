package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/42wim/cssh/device"
)

var (
	flagHost, flagCmd, flagCmdfrom, flagHostfrom, flagLogdir string
	flagPipe                                                 bool
	wg                                                       sync.WaitGroup
	cfg                                                      config
)

func init() {
	flag.StringVar(&flagHost, "host", "", "host")
	flag.StringVar(&flagCmd, "cmd", "", "single command to execute")
	flag.StringVar(&flagCmdfrom, "cmd-from", "", "file containing commands to execute")
	flag.StringVar(&flagHostfrom, "host-from", "", "file containing hosts")
	flag.StringVar(&flagLogdir, "logdir", "", "directory to log output")
	flag.BoolVar(&flagPipe, "pipe", false, "pipe to stdin")
	flag.Parse()
	if flagHost == "" && flagHostfrom == "" {
		flag.PrintDefaults()
		return
	}
	cfg = readConfig("cssh.conf")
}

func doHost(host string, cmds []string) {
	defer wg.Done()
	cred := getCredentials(host)
	if cred.Username == "" {
		log.Println("couldn't get credentials for " + host)
		return
	}
	d := &device.CiscoDevice{Hostname: host, Username: cred.Username, Password: cred.Password, Enable: cred.Enable, Logdir: flagLogdir}
	err := d.Connect()
	if err != nil {
		log.Println(err)
		return
	}
	defer d.Close()

	for _, cmd := range cmds {
		output, err := d.Cmd(cmd)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(output)
	}
	if flagPipe {
		scanner := bufio.NewScanner(os.Stdin)
		d.Echo = false
		d.EnableLog = true
		d.Cmd("")
		if d == nil {
			log.Println("something went wrong in doPipe(" + host + ")")
			return
		}
		for scanner.Scan() {
			cmd := scanner.Text()
			output, err := d.Cmd(cmd)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Print(output)
		}
	}
}

func parseFile(name string) []string {
	lines := []string{}
	dat, _ := ioutil.ReadFile(name)
	for _, line := range strings.Split(string(dat), "\n") {
		if line != "" {
			if strings.Contains(line, "#") {
			} else {
				lines = append(lines, line)
			}
		}
	}
	return lines
}

func parseFlags(filename string, stdin string) []string {
	lines := []string{}
	if filename != "" {
		lines = parseFile(filename)
	}
	if stdin != "" {
		lines = append(lines, stdin)
	}
	return lines
}

func main() {
	hosts := parseFlags(flagHostfrom, flagHost)
	cmds := parseFlags(flagCmdfrom, flagCmd)
	wg.Add(len(hosts))
	for _, host := range hosts {
		time.Sleep(time.Millisecond * 50)
		go doHost(host, cmds)
	}
	wg.Wait()
}
