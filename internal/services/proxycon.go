package services

import (
	"net/http"
	"net/url"
	"time"
)

func ProxyCon(proxyUrl string, timeout time.Duration) (*http.Client, error) {
	//creating the proxyURL
	proxyURL, err := url.Parse(proxyUrl)

	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	//adding the Transport object to the http Client
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * timeout,
	}

	return client, nil
}
