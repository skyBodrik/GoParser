package services

import (
	"golang.org/x/net/proxy"
	"log"
	"net/http"
	"time"
)

func ProxyCon(proxyAddr string, proxyAuth *proxy.Auth, timeout time.Duration) (*http.Client, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, proxyAuth, proxy.Direct)
	if err != nil {
		log.Fatalln("can't connect to the proxy:", err)
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	//adding the Transport object to the http Client
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * timeout,
	}

	return client, nil
}
