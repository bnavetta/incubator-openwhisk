package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type Params struct {
	RegistryHost string `json:"registry_host"`
	RegistryPort int    `json:"registry_port"`
	FileHost     string `json:"file_host"`
	FilePort     int    `json:"file_port"`
	ID           int    `json:"id"`
}

func printError(msg string) {
	res, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Println(string(res))
}

func getPartition(params Params) (string, error) {
	res, err := http.Get(fmt.Sprintf("http://%v:%v/sort/partition/%v", params.FileHost, params.FilePort, params.ID))
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

func connectToSorters(reg *Registry) (map[rune]net.Conn, error) {
	conns := make(map[rune]net.Conn)
	for letter := 'a'; letter <= 'b'; letter++ {
		addr, err := reg.LookupBlocking(fmt.Sprintf("%c", letter))
		if err != nil {
			return nil, err
		}

		connStr := fmt.Sprintf("%v:7171", addr)
		fmt.Printf("Connecting to %v for %c\n", connStr, letter)
		conns[letter], err = net.Dial("tcp", connStr)
		fmt.Printf("Connected! conn = %v, err = %v\n", conns[letter], err)
		if err != nil {
			return nil, err
		}
	}

	return conns, nil
}

func main() {
	go func() {
		time.Sleep(40 * time.Second)
		fmt.Println("{ \"status\": \"timed out\" }")
		os.Exit(0)
	}()

	// var params Params
	var paramMap map[string]interface{}
	json.Unmarshal([]byte(os.Args[1]), &paramMap)
	params := Params{
		RegistryHost: paramMap["registry_host"].(string),
		RegistryPort: int(paramMap["registry_port"].(float64)),
		FileHost:     paramMap["file_host"].(string),
		FilePort:     int(paramMap["file_port"].(float64)),
		ID:           int(paramMap["id"].(float64)),
	}
	fmt.Printf("%v => %v\n", os.Args[1], params)

	registry := NewRegistry(fmt.Sprintf("%v:%v", params.RegistryHost, params.RegistryPort))

	partition, err := getPartition(params)
	if err != nil {
		printError(err.Error())
		return
	}

	sorters, err := connectToSorters(&registry)
	if err != nil {
		printError(err.Error())
		return
	}

	sr := strings.NewReader(partition)
	sc := bufio.NewScanner(sr)
	for sc.Scan() {
		word := strings.ToLower(sc.Text())
		sorter := sorters[rune(word[0])]

		_, err := fmt.Fprintf(sorter, "%03d%s", len(word), word)
		if err != nil {
			printError(err.Error())
			return
		}
	}

	for letter := 'a'; letter <= 'z'; letter++ {
		_, err := io.WriteString(sorters[rune(letter)], "007DONE!!!")
		if err != nil {
			printError(err.Error())
			return
		}
		sorters[rune(letter)].Close()
	}

	fmt.Println("{\"done\": true}")
}
