/*
 * @Author: Youwei Li
 * @Date: 2021-12-27 17:22:20
 * @LastEditTime: 2021-12-27 17:56:36
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
	"time"
)

var localPort *string = flag.String("localPort", "3002", "User access address port")
var remotePort *string = flag.String("remotePort", "20012", "Communication port with client")

// Conn related to client
type client struct {
	conn net.Conn
	er   chan bool
	// Heartbeat packet channel not received
	heart chan bool
	// Not used yet!!! The original TCP connection has been connected, and the heartbeat packet is no longer required
	disheart bool
	writ     chan bool
	recv     chan []byte
	send     chan []byte
}

// Read the data sent by the client
func (self *client) read() {
	for {
		// Disconnect if there is no data transmission for 40 seconds
		self.conn.SetReadDeadline(time.Now().Add(time.Second * 40))
		var recv []byte = make([]byte, 10240)
		n, err := self.conn.Read(recv)
		if err != nil {
			//			if strings.Contains(err.Error(), "timeout") && self.disheart {
			//				fmt.Println("Two TCP connections have been made, and the server will no longer actively disconnect")
			//				self.conn.SetReadDeadline(time.Time{})
			//				continue
			//			}
			self.heart <- true
			self.er <- true
			self.writ <- true
			// fmt.Println("The information has not been transmitted for a long time, or the client has been closed. Disconnect and continue accepting the new TCP，", err)
		}
		// After receiving the heartbeat packet HH, return the reply as it is
		if recv[0] == 'h' && recv[1] == 'h' {
			self.conn.Write([]byte("hh"))
			continue
		}
		self.recv <- recv[:n]
	}
}

// Processing heartbeat packets
//func (self client) cHeart() {
//	for {
//		var recv []byte = make([]byte, 2)
//		var chanrecv []byte = make(chan []byte)
//		self.conn.SetReadDeadline(time.Now().Add(time.Second * 30))
//		n, err := self.conn.Read(recv)
//		chanrecv <- recv
//		if err != nil {
//			self.heart <- true
//			fmt.Println("Heartbeat packet timeout", err)
//			break
//		}
//		if recv[0] == 'h' && recv[1] == 'h' {
//			self.conn.Write([]byte("hh"))
//		}
//	}
//}

// Send data to client
func (self client) write() {
	for {
		var send []byte = make([]byte, 10240)
		select {
		case send = <-self.send:
			self.conn.Write(send)
		case <-self.writ:
			// fmt.Println("Write client process shutdown")
			break
		}
	}
}

// Conn related to user
type user struct {
	conn net.Conn
	er   chan bool
	writ chan bool
	recv chan []byte
	send chan []byte
}

// Read the data from the user
func (self user) read() {
	self.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 800))
	for {
		var recv []byte = make([]byte, 10240)
		n, err := self.conn.Read(recv)
		self.conn.SetReadDeadline(time.Time{})
		if err != nil {
			self.er <- true
			self.writ <- true
			// fmt.Println("Failed to read user", err)
			break
		}
		self.recv <- recv[:n]
	}
}

// Send data to user
func (self user) write() {
	for {
		var send []byte = make([]byte, 10240)
		select {
		case send = <-self.send:
			self.conn.Write(send)
		case <-self.writ:
			// fmt.Println("Write user process shutdown")
			break
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if flag.NFlag() != 2 {
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
	// Listening port
	c, err := net.Listen("tcp", ":"+*remotePort)
	log(err)
	u, err := net.Listen("tcp", ":"+*localPort)
	log(err)
	// When the first TCP is closed or TCP is established with the browser, you should return to re listen
TOP:
	// Listen for user links
	Uconn := make(chan net.Conn)
	go goaccept(u, Uconn)
	// Be sure to accept the client first
	fmt.Println("Ready to connect")
	clientconnn := accept(c)
	fmt.Println("Client connected", clientconnn.LocalAddr().String())
	recv := make(chan []byte)
	send := make(chan []byte)
	heart := make(chan bool, 1)
	// 1 position is to prevent two read threads from being stuck forever after one exits
	er := make(chan bool, 1)
	writ := make(chan bool)
	client := &client{clientconnn, er, heart, false, writ, recv, send}
	go client.read()
	go client.write()
	// You may need to deal with the heartbeat here
	for {
		select {
		case <-client.heart:
			goto TOP
		case userconnn := <-Uconn:
			// Not used yet
			client.disheart = true
			recv = make(chan []byte)
			send = make(chan []byte)
			// 1 position is to prevent two read threads from being stuck forever after one exits
			er = make(chan bool, 1)
			writ = make(chan bool)
			user := &user{userconnn, er, writ, recv, send}
			go user.read()
			go user.write()
			// When both sockets are created, enter handle processing
			go handle(client, user)
			goto TOP
		}
	}
}

// Listening port function
func accept(con net.Listener) net.Conn {
	CorU, err := con.Accept()
	logExit(err)
	return CorU
}

// Listen for port functions in another process
func goaccept(con net.Listener, Uconn chan net.Conn) {
	CorU, err := con.Accept()
	logExit(err)
	Uconn <- CorU
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
		// fmt.Printf("An error occurred and the thread exited: %v\n", err)
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

// Processing of connection between two sockets
func handle(client *client, user *user) {
	for {
		var clientrecv = make([]byte, 10240)
		var userrecv = make([]byte, 10240)
		select {
		case clientrecv = <-client.recv:
			user.send <- clientrecv
		case userrecv = <-user.recv:
			// fmt.Println("Message from browser", string(userrecv))
			client.send <- userrecv
			// User has an error. Close the sockets at both ends
		case <-user.er:
			// fmt.Println("User is closed. Close client and user")
			client.conn.Close()
			user.conn.Close()
			runtime.Goexit()
			// An error occurred in the client. Close the sockets at both ends
		case <-client.er:
			// fmt.Println("Client is closed. Close client and user")
			user.conn.Close()
			client.conn.Close()
			runtime.Goexit()
		}
	}
}
