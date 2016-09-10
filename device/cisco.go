package device

import (
	"bufio"
	"fmt"
	"github.com/ScriptRock/crypto/ssh"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
	ReadChan  chan *string
	StopChan  chan struct{}
	client    *ssh.Client
}

func (d *CiscoDevice) Connect() error {
	config := &ssh.ClientConfig{
		Timeout: time.Second * 5,
		User:    d.Username,
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
		client.Conn.Close()
		return err
	}
	d.client = client
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
	d.client.Conn.Close()
	d.session.Close()
}

func (d *CiscoDevice) Cmd(cmd string) (string, error) {
	var result string
	bufstdout := bufio.NewReader(d.stdout)
	lines := strings.Split(cmd, "!")
	for _, line := range lines {
		io.WriteString(d.stdin, line+"\n")
		time.Sleep(time.Millisecond * 100)
	}
	go d.readln(bufstdout)
	for {
		select {
		case output := <-d.ReadChan:
			{
				if output == nil {
					continue
				}
				if d.Echo == false {
					result = strings.Replace(*output, lines[0], "", 1)
				} else {
					result = *output
				}
				return result, nil
			}
		case <-d.StopChan:
			{
				if d.session != nil {
					d.session.Close()
				}
				d.client.Conn.Close()
				d.Close()
				return "", fmt.Errorf("EOF")
			}
		case <-time.After(time.Second * 30):
			{
				fmt.Println("timeout on", d.Hostname)
				if d.session != nil {
					d.session.Close()
				}
				d.client.Conn.Close()
				d.Connect()
				return "", nil
			}
		}
	}
}

func (d *CiscoDevice) init() {
	d.ReadChan = make(chan *string, 20)
	d.StopChan = make(chan struct{})
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
	// sometimes using conf t makes the (config-xx-something) so long that only 10 chars of
	// original prompt remain
	if len(d.Prompt) > 10 {
		d.Prompt = d.Prompt[:10]
	}
}

func (d *CiscoDevice) readln(r io.Reader) {
	//re := regexp.MustCompile(".*?#.?$")
	var re *regexp.Regexp
	if d.Prompt == "" {
		re = regexp.MustCompile("[[:alnum:]]#.?$")
	} else {
		re = regexp.MustCompile(d.Prompt + ".*?#.?$")
	}
	//fmt.Println("using prompt" + d.Prompt)
	buf := make([]byte, 10000)
	loadStr := ""
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Println("ERROR ", err)
			}
			close(d.StopChan)
		}
		loadStr += string(buf[:n])
		// logging to file if necessary
		if d.Logdir != "" {
			if d.EnableLog {
				fmt.Fprint(d.Log, string(buf[:n]))
				//loadStr = ""
			}
		}
		if re.MatchString(string(buf[:n])) {
			break
		}
		// keepalive
		d.ReadChan <- nil
	}
	loadStr = strings.Replace(loadStr, "\r", "", -1)
	d.ReadChan <- &loadStr
}
