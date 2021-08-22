package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/andrewchambers/go-fast-collectd"
)

var (
	username = flag.String("username", "", "collectd username")
	password = flag.String("password", "", "collectd Password")
	mode     = flag.String("mode", "plain-text", "Mode, one of 'plain-text', 'sign', 'encrypt'")
)

func main() {

	flag.Parse()

	opts := collectd.UDPClientOptions{
		Username: *username,
		Password: *password,
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

	total_alloc := &collectd.Metric{
		Plugin:       "golang",
		Type:         "counter",
		TypeInstance: "go-memstats-total-alloc",
		ValueTypes: []collectd.ValueType{
			collectd.COUNTER,
		},
		Interval: interval,
	}

	total_alloc.FillDefaultsAndValidate()

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
