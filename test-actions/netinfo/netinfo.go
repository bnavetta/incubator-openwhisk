package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

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

	return "unknown"
}

func getInfo() map[string]interface{} {
	interfaces, err := net.Interfaces()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	res := make([]map[string]interface{}, 0)

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()

		if err != nil {
			return map[string]interface{}{"error": err.Error()}
		}

		addrNames := make([]string, 0)
		netNames := make([]string, 0)
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPAddr:
				addrNames = append(addrNames, v.IP.String())
			case *net.IPNet:
				netNames = append(netNames, v.String())
			}
		}

		res = append(res, map[string]interface{}{
			"index":        iface.Index,
			"name":         iface.Name,
			"flags":        iface.Flags,
			"mtu":          iface.MTU,
			"hardwareAddr": iface.HardwareAddr,
			"addresses":    addrNames,
			"subnet":       netNames,
		})
	}

	return map[string]interface{}{
		"interfaces": res,
		"overlayIP":  getOverlayIP(),
	}
}

func main() {
	info := getInfo()
	res, _ := json.Marshal(info)

	time.Sleep(30 * time.Second)

	fmt.Println(string(res))
}
