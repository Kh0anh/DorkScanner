package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Kh0anh/DorkScanner/proxy"
	"github.com/Kh0anh/DorkScanner/utils"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var (
	dorksFile, proxysFile string
	checkProxy, useProxy  bool
	dorks                 *[]string
	proxys                []*proxy.Proxy
)

type counter struct {
	count int
	total int
	lock  sync.Mutex
}

type aProxy struct {
	use   bool
	proxy *proxy.Proxy
}

func init() {
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "01-02|15:04:05",
		LogFormat:       "%lvl% [%time%] %msg%\n",
	})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)

	flag.StringVar(&dorksFile, "d", "", "")
	flag.StringVar(&dorksFile, "dorks", "", "")

	flag.StringVar(&proxysFile, "p", "", "")
	flag.StringVar(&proxysFile, "proxys", "", "")

	flag.BoolVar(&checkProxy, "c", false, "")
	flag.BoolVar(&checkProxy, "check", false, "")

	flag.Parse()

	help := []string{
		"Options:",
		"    -d, --dorks <dorks.txt>         Search query",
		"    -p, --proxys <proxys.txt>       Use proxys",
		"    -c, --check                     Proxy checking",
		"\n"}
	banner := []string{
		"  _____             _       _____                                 ",
		" |  __ \\           | |     / ____|                                ",
		" | |  | | ___  _ __| | __ | (___   ___ __ _ _ __  _ __   ___ _ __ ",
		" | |  | |/ _ \\| '__| |/ /  \\___ \\ / __/ _` | '_ \\| '_ \\ / _ \\ '__|",
		" | |__| | (_) | |  |   <   ____) | (_| (_| | | | | | | |  __/ |   ",
		" |_____/ \\___/|_|  |_|\\_\\ |_____/ \\___\\__,_|_| |_|_| |_|\\___|_|   ",
		"                                                         by Kh0anh",
	}

	utils.Show(banner)
	utils.Show(help)
}
func main() {
	if dorksFile == "" {
		log.Error("-d, --dorks is required")
	}

	dorks, err := utils.ReadFileByLine(dorksFile)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if proxysFile != "" && checkProxy {
		tempProxysStr, err := utils.ReadFileByLine(proxysFile)
		if err != nil {
			log.Error(err.Error())
			return
		}
		useProxy = true

		var tempProxys []*proxy.Proxy
		for _, strProxy := range *tempProxysStr {
			tempProxy, err := proxy.NewProxy(strProxy, proxy.HTTP)
			if err != nil {
				log.Error(err)
			}
			tempProxys = append(tempProxys, tempProxy)
		}

		var wg sync.WaitGroup
		done := false
		c := counter{count: 0, total: len(tempProxys)}

		for i := 1; i <= 100; i++ {
			wg.Add(1)
			go checkProxys(tempProxys, &proxys, &c, &wg)
		}

		go func(d *bool) {
			wg.Wait()
			*d = true
		}(&done)

		for !done {
			log.Info(fmt.Sprintf("Checking proxy %d/%d", c.count, c.total))
			time.Sleep(2 * time.Second)
		}
	} else if proxysFile != "" {
		tempProxysStr, err := utils.ReadFileByLine(proxysFile)
		if err != nil {
			log.Error(err.Error())
			return
		}
		useProxy = true

		for _, strProxy := range *tempProxysStr {
			tempProxy, err := proxy.NewProxy(strProxy, proxy.HTTP)
			if err != nil {
				log.Error(err)
			}
			proxys = append(proxys, tempProxy)
		}
	}

	var results []string
	var _proxys []aProxy
	for _, proxy := range proxys {
		_proxys = append(_proxys, aProxy{proxy: proxy, use: false})
	}
	var wg sync.WaitGroup
	done := false
	c := counter{count: 0, total: len(*dorks)}

	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go dorkScan(&results, dorks, &_proxys, &c, &wg)
	}

	go func(d *bool) {
		wg.Wait()
		*d = true
	}(&done)

	for !done {
		log.Info(fmt.Sprintf("Dorks scanner %d/%d", c.count, c.total))
		time.Sleep(2 * time.Second)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func checkProxys(tempProxys []*proxy.Proxy, proxys *[]*proxy.Proxy, c *counter, wg *sync.WaitGroup) {
	defer wg.Done()

	counting := func() int {
		c.lock.Lock()
		defer c.lock.Unlock()
		temp := c.count
		c.count++
		return temp
	}

	for c.count < c.total {
		n := counting()
		if n >= c.total {
			break
		}

		proxy := tempProxys[n]
		if proxy.Check() {
			*proxys = append(*proxys, proxy)
		}
	}
}

func dorkScan(results *[]string, dorks *[]string, proxys *[]aProxy, c *counter, wg *sync.WaitGroup) {
	defer wg.Done()

	counting := func() int {
		c.lock.Lock()
		defer c.lock.Unlock()
		temp := c.count
		c.count++
		return temp
	}

	getProxy := func() *proxy.Proxy {
		c.lock.Lock()
		defer c.lock.Unlock()
		for _, proxy := range *proxys {
			if !proxy.use {
				proxy.use = true
				return proxy.proxy
			}
		}
		return nil
	}

	var proxy *proxy.Proxy

	for c.count < c.total {
		n := counting()
		if n >= c.total {
			break
		}
		dork := (*dorks)[n]
		if proxy == nil {
			proxy = getProxy()
		}
		var _results *[]string
		var lock sync.Mutex
		i := 0
		if !dorkScanWorker(_results, proxy, dork, &i, &lock) {
			proxy = nil
			continue
		}
		func() {
			c.lock.Lock()
			defer c.lock.Unlock()
			*results = append(*results, *_results...)
		}()
	}
}

func dorkScanWorker(results *[]string, proxy *proxy.Proxy, dork string, index *int, lock *sync.Mutex) bool {
	_dork := strings.Replace(dork, " ", "%20", -1)
	_url := strings.Replace(strings.Replace("https://www.google.com/search?q={dork}&start={index}&client=firefox-b-e", "{dork}", _dork, -1), "{index}", strconv.Itoa(*index), -1)
	fmt.Println(_url)
	response, err := proxy.Get(_url)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if response.Request.URL.String() == "https://www.google.com/sorry/index" {
		fmt.Println("BAN")
		return false
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println(string(body))
	pattern := `<a\s+href="\/url\?q=([^"&]+)&amp;sa=U&amp;ved=([^"&]+)&amp;usg=([^"&]+)"\s+data-ved="([^"]+)"`
	regex := regexp.MustCompile(pattern)
	j := 0
	for _, group := range regex.FindAllStringSubmatch(string(body), -1) {
		func() {
			lock.Lock()
			defer lock.Unlock()
			fmt.Println(group[1])
			*results = append(*results, group[1])
			j++
		}()
	}
	if j < 9 {
		return true
	}
	*index = *index + 10
	return dorkScanWorker(results, proxy, dork, index, lock)
}
