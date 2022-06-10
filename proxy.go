package main

import (
	"log"

	"golang.org/x/sys/windows/registry"
)

func (client *FNRadioClient) setupSystemProxy() {
	k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		panic(err)
	}

	client.previousProxyEnabled, _, _ = k.GetIntegerValue("ProxyEnable")
	client.previousProxyServer, _, _ = k.GetStringValue("ProxyServer")

	if client.previousProxyEnabled == 1 {
		if client.previousProxyServer == "https=127.0.0.1:18149" {
			client.alreadyProxying = true
			client.previousProxyEnabled = 0
			client.previousProxyServer = ""
		} else {
			log.Println("WARN: This tool doesn't work with existing proxies")
		}
	}

	err = k.SetDWordValue("ProxyEnable", 1)
	if err != nil {
		panic(err)
	}

	err = k.SetStringValue("ProxyServer", "https=127.0.0.1:18149")
	if err != nil {
		panic(err)
	}

	_ = k.Close()
}

func (client *FNRadioClient) revertSystemProxy() {
	k, _ := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	_ = k.SetDWordValue("ProxyEnable", uint32(client.previousProxyEnabled))

	if client.previousProxyServer != "" {
		_ = k.SetStringValue("ProxyServer", client.previousProxyServer)
	} else {
		_ = k.DeleteValue("ProxyServer")
	}

	_ = k.Close()
}
