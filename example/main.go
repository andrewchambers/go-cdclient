package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/andrewchambers/go-cdclient"
)

var (
	authfile = flag.String("authfile", "./collectd.auth", "collectd Password")
	username = flag.String("username", "metrics", "collectd auth username")
	mode     = flag.String("mode", "encrypt", "Mode, one of 'plain-text', 'sign', 'encrypt'")
)

func main() {

	flag.Parse()

	auth, err := cdclient.NewAuthFile(*authfile)
	if err != nil {
		panic(err)
	}

	password, ok := auth.Password(*username)
	if !ok {
		panic(fmt.Sprintf("no password for: %s", *username))
	}

	opts := cdclient.UDPClientOptions{
		Username: *username,
		Password: password,
	}

	switch *mode {
	case "plain-text":
		opts.Mode = cdclient.UDPPlainText
	case "sign":
		opts.Mode = cdclient.UDPSign
	case "encrypt":
		opts.Mode = cdclient.UDPEncrypt
	default:
		panic("invalid -mode")
	}

	c, err := cdclient.DialUDP("127.0.0.1:25826", opts)
	if err != nil {
		panic(err)
	}

	memStats := runtime.MemStats{}
	interval := 1 * time.Second
	host, _ := os.Hostname()

	total_alloc := &cdclient.Metric{
		Host:           host,
		Plugin:         "go",
		PluginInstance: path.Base(os.Args[0]),
		Type:           "counter",
		TypeInstance:   "go-memstats-total-alloc",
		DSTypes: []cdclient.DSType{
			cdclient.COUNTER,
		},
		Interval: interval,
	}

	total_alloc.Validate()

	for {
		runtime.ReadMemStats(&memStats)
		err := c.AddValues(
			total_alloc, time.Now(), float64(memStats.TotalAlloc),
		)
		if err != nil {
			panic(err)
		}
		err = c.Flush()
		if err != nil {
			panic(err)
		}
		time.Sleep(interval)
	}

}
