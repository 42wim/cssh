package device

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ScriptRock/crypto/ssh"
)

type CiscoDevice struct {
	Username  string
	Password  string
	Enable    string
	name      string
	Hostname  string
	stdin     io.WriteCloser
	stdout    io.Reader
	session   *ssh.Session
	Echo      bool
	EnableLog bool
	Logdir    string
	Log       *os.File
	Prompt    string
}

func (d *CiscoDevice) Connect() error {
	config := &ssh.ClientConfig{
		User: d.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(d.Password),
		},
		Config: ssh.Config{
			Ciphers: ssh.AllSupportedCiphers(),
		},
	}
	client, err := ssh.Dial("tcp", d.Hostname+":22", config)
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	d.stdin, _ = session.StdinPipe()
	d.stdout, _ = session.StdoutPipe()
	d.Echo = true
	d.EnableLog = true
	modes := ssh.TerminalModes{
		ssh.ECHO:          0, // disable echoing
		ssh.OCRNL:         0,
		ssh.TTY_OP_ISPEED: 38400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 38400, // output speed = 14.4kbaud
	}
	session.RequestPty("vt100", 0, 2000, modes)
	session.Shell()
	if d.Logdir != "" {
		t := time.Now()
		d.Log, err = os.OpenFile(filepath.Join(d.Logdir, t.Format("200601021504")+"-"+d.Hostname), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
	}
	d.init()
	d.session = session
	return nil
}

func (d *CiscoDevice) Close() {
	d.session.Close()
}

func (d *CiscoDevice) Cmd(cmd string) (string, error) {
	bufstdout := bufio.NewReader(d.stdout)
	lines := strings.Split(cmd, "!")
	for _, line := range lines {
		io.WriteString(d.stdin, line+"\n")
		time.Sleep(time.Millisecond * 100)
	}
	output, err := d.readln(bufstdout)
	if err != nil {
		return "", err
	}
	output = strings.Replace(output, "\r", "", -1)
	if d.Echo == false {
		output = strings.Replace(output, lines[0], "", 1)
	}
	if d.Logdir != "" {
		return "", nil
	}
	return output, nil
}

func (d *CiscoDevice) init() {
	bufstdout := bufio.NewReader(d.stdout)
	io.WriteString(d.stdin, "enable\n")
	time.Sleep(time.Millisecond * 100)
	re := regexp.MustCompile("assword:")
	buf := make([]byte, 1000)
	loadStr := ""
	for {
		n, err := bufstdout.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		loadStr += string(buf[:n])
		if re.MatchString(loadStr) {
			io.WriteString(d.stdin, d.Enable+"\n")
			break
		} else {
			break
		}
	}
	d.Cmd("terminal length 0")
	d.Cmd("")
	prompt, _ := d.Cmd("")
	d.Prompt = strings.TrimSpace(prompt)
	d.Prompt = strings.Replace(d.Prompt, "#", "", -1)
}

func (d *CiscoDevice) readln(r *bufio.Reader) (string, error) {
	var re *regexp.Regexp
	if d.Prompt == "" {
		re = regexp.MustCompile("[[:alnum:]]#.?$")
	} else {
		re = regexp.MustCompile(d.Prompt + ".*?#$")
	}
	buf := make([]byte, 10000)
	loadStr := ""
	for {
		n, err := r.Read(buf)
		if err != nil {
			return "", err
		}
		loadStr += string(buf[:n])
		// logging to file if necessary
		if d.Logdir != "" {
			if d.EnableLog {
				fmt.Fprint(d.Log, string(buf[:n]))
			}
		}
		if re.MatchString(string(buf[:n])) {
			break
		}
	}
	return loadStr, nil
}
