package proxy

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Proxy struct {
	proxyType int
	httpProxy *url.URL
}

var (
	HTTP = 0
)

func NewProxy(proxyURL string, proxyType int) (*Proxy, error) {
	if strings.HasPrefix(proxyURL, "http://") {
		proxyURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		return &Proxy{httpProxy: proxyURL, proxyType: proxyType}, nil
	} else if regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+$`).MatchString(proxyURL) {
		proxyURL, err := url.Parse("http://" + proxyURL)
		if err != nil {
			return nil, err
		}
		return &Proxy{httpProxy: proxyURL, proxyType: proxyType}, nil
	}

	return nil, errors.New("proxy type error \"" + proxyURL + "\"")
}

func (_proxy Proxy) getHTTP(target string) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(_proxy.httpProxy),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(target)
	if err != nil {
		return resp, err
	}

	return resp, err
}

func (_proxy Proxy) postHTTP(target string, data []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", target, bytes.NewBuffer(data))

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (_proxy Proxy) Check() bool {
	resp, err := _proxy.Get("https://www.google.com")

	if err != nil {
		return false
	}
	body, err := ioutil.ReadAll(resp.Body)

	if strings.Contains(string(body), "drive.google.com") {
		return true
	}
	return false
}
func (_proxy Proxy) Get(target string) (*http.Response, error) {
	if _proxy.proxyType == 0 {
		return _proxy.getHTTP(target)
	}
	return nil, errors.New("non proxy")
}

func (_proxy Proxy) Post(target string, data []byte) (*http.Response, error) {
	if _proxy.proxyType == 0 {
		return _proxy.postHTTP(target, data)
	}
	return nil, errors.New("non proxy")
}
