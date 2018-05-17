package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

func main() {
	go func() {
		time.Sleep(40 * time.Second)
		fmt.Println("{ \"status\": \"timed out\" }")
		os.Exit(0)
	}()

	var params Params
	json.Unmarshal([]byte(os.Args[1]), &params)

	registry := NewRegistry(params.Registry)
	pongAddr, err := registry.LookupBlocking("pong")
	if err != nil {
		printError(err.Error())
		return
	}

	conn, err := net.Dial("tcp", pongAddr+":1234")
	if err != nil {
		printError(err.Error())
		return
	}
	defer conn.Close()

	_, err = io.WriteString(conn, "ping")
	if err != nil {
		printError(err.Error())
		return
	}

	fmt.Println("Sent ping")

	response, err := ioutil.ReadAll(conn)
	if err != nil {
		printError(err.Error())
		return
	}

	fmt.Println("Received response: " + string(response))

	j, _ := json.Marshal(map[string]string{"response": string(response)})
	fmt.Println(j)
}
