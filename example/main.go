package main

import (
	"os"
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/andrewchambers/go-fast-collectd"
)

var (
	authfile = flag.String("authfile", "./collectd.auth", "collectd Password")
	username = flag.String("username", "metrics", "collectd auth username")
	mode     = flag.String("mode", "encrypt", "Mode, one of 'plain-text', 'sign', 'encrypt'")
)

func main() {

	flag.Parse()

	auth, err := collectd.NewAuthFile(*authfile)
	if err != nil {
		panic(err)
	}

	password, ok := auth.Password(*username)
	if !ok {
		panic(fmt.Sprintf("no password for: %s", *username))
	}

	opts := collectd.UDPClientOptions{
		Username: *username,
		Password: password,
	}

	switch *mode {
	case "plain-text":
		opts.Mode = collectd.UDPPlainText
	case "sign":
		opts.Mode = collectd.UDPSign
	case "encrypt":
		opts.Mode = collectd.UDPEncrypt
	default:
		panic("invalid -mode")
	}

	c, err := collectd.DialUDP("127.0.0.1:25826", opts)
	if err != nil {
		panic(err)
	}

	memStats := runtime.MemStats{}
	interval := 1 * time.Second
	host, _ := os.Hostname()

	total_alloc := &collectd.Metric{
		Host:           host,
		Plugin:         "golang",
		PluginInstance: os.Args[0],
		Type:           "counter",
		TypeInstance:   "go-memstats-total-alloc",
		ValueTypes: []collectd.ValueType{
			collectd.COUNTER,
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
