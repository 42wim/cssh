# cssh
Tool to connect to (recent) Cisco switches and accesspoints and execute commands. 

Tested on:
- Cisco Nexus devices (n2k/n5k/n7k/n9k)
- A lot of Cisco Catalyst types (2960, 3750, ..)
- A lot of Cisco Aironet APs

## use cases
* configure a lot of devices easy and at the same time
* get realtime output from a specified list of switches (e.g searching a mac address)
* run simple commands without an interactive shell

## building
Make sure you have [Go](https://golang.org/doc/install) properly installed, including setting up your [GOPATH](https://golang.org/doc/code.html#GOPATH)

Next, run

 ```
 $ cd $GOPATH
 $ go get github.com/42wim/cssh
 ```

 You'll have the binary 'cssh' in $GOPATH/bin

## usage

```
$ ./cssh
  -cmd="": single command to execute
  -cmd-from="": file containing commands to execute
  -host="": host
  -host-from="": file containing hosts
  -logdir="": directory to log output
  -pipe=false: pipe to stdin
```


## config
cssh looks for cssh.conf in current directory.  
This config file is necessary because it contains the credentials.  

Format is 
```
[device "yourswitch"] #yourswitch can be a FQDN or an IP or an regular expression
username=admin 
password=abc
enable=abc
```

E.g.
```
[device "switch-building-x-*"]
username=admin
password=abc
enable=abc
```

Also see cssh.conf.dist

## examples

### using -pipe
```
$ echo "sh version"| cssh -host lab-switch-1 -pipe|grep uptime
lab-switch-1 uptime is 20 weeks, 4 days, 9 hours, 19 minutes
```

### using -logdir
```
$ mkdir logs
$ cssh -host lab-switch-1 -cmd "sh tech" -logdir logs
$ ls -b logs/
201503242329-lab-switch-1
```

### interactive commands
Use the '!' separator

```
$ cssh -cmd 'ping!ip!4.4.4.4!5!100!2!n!n' -host lab-switch-1
Protocol [ip]: ip
Target IP address: 4.4.4.4
Repeat count [5]: 5
Datagram size [100]: 100
Timeout in seconds [2]: 2
Extended commands [n]: n
Sweep range of sizes [n]: n
Type escape sequence to abort.
Sending 5, 100-byte ICMP Echos to 4.4.4.4, timeout is 2 seconds:
.....
Success rate is 0 percent (0/5)
lab-switch-1#
```

```
$ cssh -cmd 'copy run start!' -host lab-switch-1
copy run start
Destination filename [startup-config]?
Building configuration...
[OK]
lab-switch-1#
```

### 1 host and 1 command
```
$ cssh -host lab-switch-1 -cmd "sh clock"
sh clock
23:41:27.572 CET Wed Mar 11 2015
lab-switch-1#
```

### 1 host and multiple commands
see examples/cmd-configexample for the contents

```
$ cssh -host lab-switch-1 -cmd-from cmd-configexample
conf t
Enter configuration commands, one per line.  End with CNTL/Z.
lab-switch-1(config)#interface GigabitEthernet0/2
lab-switch-1(config-if)#switchport trunk allowed vlan 234,336,337,356,445,488
lab-switch-1(config-if)#switchport mode trunk
lab-switch-1(config-if)#switchport nonegotiate
lab-switch-1(config-if)#switchport protected
lab-switch-1(config-if)#storm-control broadcast level 50.00 30.00
lab-switch-1(config-if)#storm-control action trap
lab-switch-1(config-if)#no cdp enable
lab-switch-1(config-if)#spanning-tree portfast
lab-switch-1(config-if)#spanning-tree bpduguard enable
lab-switch-1(config-if)#ip dhcp snooping limit rate 30
lab-switch-1(config-if)#exit
lab-switch-1(config)#exit
lab-switch-1#
```

### multiple hosts and 1 command
labswitches contains  
```
lab-switch-1
lab-switch-2
```

```
$ cssh -host-from labswitches -cmd "sh clock"
sh clock
*22:45:16.946 UTC Wed Mar 11 2015
lab-switch-1#
sh clock
23:45:18.604 CET Wed Mar 11 2015
lab-switch-2#
```

### multiple hosts and multiple commands
left as an exercise for the reader
