/*
 * @Author: Youwei Li
 * @Date: 2021-12-27 17:22:26
 * @LastEditTime: 2021-12-27 17:57:19
 * @LastEditors: Youwei Li
 * @Description:
 */
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var host *string = flag.String("host", "127.0.0.1", "Please enter the server IP")
var remotePort *string = flag.String("remotePort", "20012", "Server address port")
var localPort *string = flag.String("localPort", "80", "Local port")

// Browser related conn
type browser struct {
	conn net.Conn
	er   chan bool
	writ chan bool
	recv chan []byte
	send chan []byte
}

// Read the data from the browser
func (self browser) read() {
	for {
		var recv []byte = make([]byte, 10240)
		n, err := self.conn.Read(recv)
		if err != nil {
			self.writ <- true
			self.er <- true
			// fmt.Println("Failed to read browser", err)
			break
		}
		self.recv <- recv[:n]
	}
}

// Send data to browser
func (self browser) write() {
	for {
		var send []byte = make([]byte, 10240)
		select {
		case send = <-self.send:
			self.conn.Write(send)
		case <-self.writ:
			// fmt.Println("Write browser process shutdown")
			break
		}
	}
}

// Conn related to server
type server struct {
	conn net.Conn
	er   chan bool
	writ chan bool
	recv chan []byte
	send chan []byte
}

// Read the data from the server
func (self *server) read() {
	// Isheart and timeout jointly judge whether the setreaddeadline is set by themselves
	var isheart bool = false
	// Send a heartbeat packet every 20 seconds
	self.conn.SetReadDeadline(time.Now().Add(time.Second * 20))
	for {
		var recv []byte = make([]byte, 10240)
		n, err := self.conn.Read(recv)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") && !isheart {
				// fmt.Println("Send heartbeat packet")
				self.conn.Write([]byte("hh"))
				// 4 second cardio Jump Pack
				self.conn.SetReadDeadline(time.Now().Add(time.Second * 4))
				isheart = true
				continue
			}
			// The browser may disconnect without sending a message. At this time, a 0 will be sent in order to always have a TCP path with the server
			self.recv <- []byte("0")
			self.er <- true
			self.writ <- true
			// fmt.Println("If no heartbeat packet is received or the server is shut down, close this TCP message", err)
			break
		}
		// Received heartbeat packet
		if recv[0] == 'h' && recv[1] == 'h' {
			// fmt.Println("Received heartbeat packet")
			self.conn.SetReadDeadline(time.Now().Add(time.Second * 20))
			isheart = false
			continue
		}
		self.recv <- recv[:n]
	}
}

// Send data to server
func (self server) write() {
	for {
		var send []byte = make([]byte, 10240)
		select {
		case send = <-self.send:
			self.conn.Write(send)
		case <-self.writ:
			// fmt.Println("Write server process shutdown")
			break
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if flag.NFlag() != 3 {
		flag.PrintDefaults()
		os.Exit(1)
	}
	local, _ := strconv.Atoi(*localPort)
	remote, _ := strconv.Atoi(*remotePort)
	if !(local >= 0 && local < 65536) {
		fmt.Println("Port setting error")
		os.Exit(1)
	}
	if !(remote >= 0 && remote < 65536) {
		fmt.Println("Port setting error")
		os.Exit(1)
	}
	target := net.JoinHostPort(*host, *remotePort)
	for {
		// Link port
		serverconn := dail(target)
		recv := make(chan []byte)
		send := make(chan []byte)
		// 1 position is to prevent two read threads from being stuck forever after one exits
		er := make(chan bool, 1)
		writ := make(chan bool)
		next := make(chan bool)
		server := &server{serverconn, er, writ, recv, send}
		go server.read()
		go server.write()
		go handle(server, next)
		<-next
	}
}

// Display error
func log(err error) {
	if err != nil {
		fmt.Printf("An error occurred: %v\n", err)
	}
}

// Show errors and exit
func logExit(err error) {
	if err != nil {
		fmt.Printf("An error occurred and the thread exited: %v\n", err)
		runtime.Goexit()
	}
}

// Displays the error and closes the link, exiting the thread
func logClose(err error, conn net.Conn) {
	if err != nil {
		// fmt.Println("The other party has been closed", err)
		runtime.Goexit()
	}
}

// Link port
func dail(hostport string) net.Conn {
	conn, err := net.Dial("tcp", hostport)
	logExit(err)
	return conn
}

// Processing of connection between two sockets
func handle(server *server, next chan bool) {
	var serverrecv = make([]byte, 10240)
	// Block here, wait for data from the server, and then link to the browser
	fmt.Println("Wait for a message from the server")
	serverrecv = <-server.recv
	// Connect, the next TCP connects to the server
	next <- true
	// fmt.Println("Start a new TCP link, and the message is:", string(serverrecv))
	var browse *browser
	// The server sends data and links to the local port 80
	serverconn := dail("127.0.0.1:" + *localPort)
	recv := make(chan []byte)
	send := make(chan []byte)
	er := make(chan bool, 1)
	writ := make(chan bool)
	browse = &browser{serverconn, er, writ, recv, send}
	go browse.read()
	go browse.write()
	browse.send <- serverrecv

	for {
		var serverrecv = make([]byte, 10240)
		var browserrecv = make([]byte, 10240)
		select {
		case serverrecv = <-server.recv:
			if serverrecv[0] != '0' {
				browse.send <- serverrecv
			}
		case browserrecv = <-browse.recv:
			server.send <- browserrecv
		case <-server.er:
			// fmt.Println("Server is closed. Close server and browse")
			server.conn.Close()
			browse.conn.Close()
			runtime.Goexit()
		case <-browse.er:
			// fmt.Println("Browse is closed. Close the server and browse")
			server.conn.Close()
			browse.conn.Close()
			runtime.Goexit()
		}
	}
}
