package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

var (
	dataPath  = "/projects/exo/expl.speck"
	info_log  *log.Logger
	error_log *log.Logger
)

func log_error(template string, args ...interface{}) {
	error_log.Printf(template, args...)
}

func log_info(template string, args ...interface{}) {
	info_log.Printf(template, args...)
}

func bail(status int, template string, args ...interface{}) {
	if status == 0 {
		fmt.Fprintf(os.Stdout, template, args...)
	} else {
		fmt.Fprintf(os.Stderr, template, args...)
	}
	os.Exit(status)
}

func handleConnection(conn *Connection) {
	defer conn.Close()
	conn.Login()

	system, err := randomSystem()
	if err != nil {
		log_error("player %s failed to get random system: %v", conn.PlayerName(), err)
		return
	}
	system.Arrive(conn)
	if system.planets == 1 {
		fmt.Fprintf(conn, "you are in the system %s. There is %d planet here.\n", system.name, system.planets)
	} else {
		fmt.Fprintf(conn, "you are in the system %s. There are %d planets here.\n", system.name, system.planets)
	}
	for {
		line, err := conn.ReadString('\n')
		switch err {
		case io.EOF:
			return
		case nil:
			break
		default:
			log_error("failed to read line from player %s: %v", conn.PlayerName(), err)
			return
		}
		line = strings.TrimSpace(line)

		if conn.IsMining() {
			conn.StopMining()
		}

		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")

		if isCommand(parts[0]) {
			runCommand(conn, parts[0], parts[1:]...)
			continue
		}

		switch parts[0] {
		case "quit":
			return
		default:
			fmt.Fprintf(conn, "hmm I'm not sure I know that one.\n")
		}
	}
}

func main() {
	dbconnect()
	rand.Seed(time.Now().UnixNano())
	info_log = log.New(os.Stdout, "[INFO] ", 0)
	error_log = log.New(os.Stderr, "[ERROR] ", 0)

	setupDb()
	listener, err := net.Listen("tcp", ":9220")
	if err != nil {
		bail(E_No_Port, "unable to start server: %v", err)
	}
	go RunQueue()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log_error("error accepting connection: %v", err)
			continue
		}
		go handleConnection(NewConnection(conn))
	}
}
