package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type Params struct {
	Registry string
}

func printError(msg string) {
	res, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Println(string(res))
}

func getOverlayIP() string {
	// TODO: assuming eth1 might be too fragile, could do a subnet check
	iface, err := net.InterfaceByName("eth1")
	if err != nil {
		return err.Error()
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return err.Error()
	}

	for _, addr := range addrs {
		switch a := addr.(type) {
		case *net.IPAddr:
			return a.IP.String()
		case *net.IPNet:
			return a.IP.String()
		default:
			continue
		}
	}

	return ""
}

func main() {
	go func() {
		time.Sleep(40 * time.Second)
		fmt.Println("{ \"status\": \"timed out\" }")
		os.Exit(0)
	}()

	var params Params
	json.Unmarshal([]byte(os.Args[1]), &params)

	registry := NewRegistry(params.Registry)

	addr := getOverlayIP()
	err := registry.Register("pong", addr)
	if err != nil {
		printError(err.Error())
		return
	}

	listener, err := net.Listen("tcp", addr+":1234")
	fmt.Println("Listening on " + addr + ":1234")
	if err != nil {
		printError(err.Error())
		return
	}
	defer listener.Close()

	conn, err := listener.Accept()
	fmt.Printf("Connection from %v\n", conn.RemoteAddr().String())
	if err != nil {
		printError(err.Error())
		return
	}

	io.WriteString(conn, "pong")
	time.Sleep(2 * time.Second)
	conn.Close()

	j, _ := json.Marshal(map[string]string{"remote": conn.RemoteAddr().String()})
	fmt.Println(string(j))
}
