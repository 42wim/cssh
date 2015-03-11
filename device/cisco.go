package device

import (
	"bufio"
	"github.com/ScriptRock/crypto/ssh"
	"io"
	"log"
	"regexp"
	"strings"
	"time"
)

type CiscoDevice struct {
	Username string
	Password string
	Enable   string
	name     string
	Hostname string
	stdin    io.WriteCloser
	stdout   io.Reader
	session  *ssh.Session
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
	modes := ssh.TerminalModes{
		ssh.ECHO:          0, // disable echoing
		ssh.OCRNL:         0,
		ssh.TTY_OP_ISPEED: 38400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 38400, // output speed = 14.4kbaud
	}
	session.RequestPty("vt100", 0, 2000, modes)
	session.Shell()
	d.init()
	d.session = session
	return nil
}

func (d *CiscoDevice) Close() {
	d.session.Close()
}

func (d *CiscoDevice) Cmd(cmd string) string {
	bufstdout := bufio.NewReader(d.stdout)
	io.WriteString(d.stdin, cmd+"\n")
	time.Sleep(time.Millisecond * 100)
	return strings.Replace(d.readln(bufstdout), "\r", "", -1)
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
}

func (d *CiscoDevice) readln(r *bufio.Reader) string {
	re := regexp.MustCompile(".*?#.?$")
	buf := make([]byte, 1000)
	loadStr := ""
	for {
		n, err := r.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		loadStr += string(buf[:n])
		if re.MatchString(loadStr) {
			break
		}
	}
	return loadStr
}
