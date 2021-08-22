// A fast package for sending metrics to collectd.
package collectd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type ValueType byte

const (
	COUNTER  = 0
	GAUGE    = 1
	DERIVE   = 2
	ABSOLUTE = 3
)

type Metric struct {
	Host           string
	Plugin         string
	PluginInstance string
	Type           string
	TypeInstance   string
	ValueTypes     []ValueType
	Interval       time.Duration
}

type ValueList struct {
	Metric *Metric
	Time   time.Time
	Values []float64
}

// Default plugin, used if a metric does not specify it.
var DefaultPlugin string = "golang"

func (m *Metric) FillDefaultsAndValidate() {
	if m.Interval == 0 {
		m.Interval = 10 * time.Second
	}
	if m.Host == "" {
		host, _ := os.Hostname()
		m.Host = host
	}
	if m.Plugin == "" {
		m.Plugin = DefaultPlugin
	}
	for i, s := range []string{
		m.Host,
		m.Plugin,
		m.PluginInstance,
		m.Type,
		m.TypeInstance,
	} {
		if len(s) > 63 {
			panic(fmt.Sprintf("bad metric identifier: %q is too long", s))
		}
		var forbidden []rune
		if i == 1 || i == 3 {
			// Plugin and Type cannot contain '-'
			forbidden = []rune{'\\', '/', '"', '-', 0}
		} else {
			forbidden = []rune{'\\', '/', '"', 0}
		}
		if i != 2 && i != 4 {
			if len(s) == 0 {
				panic(fmt.Sprintf("bad metric, mandatory field empty"))
			}
		}
		for _, f := range forbidden {
			if strings.IndexRune(s, f) != -1 {
				panic(fmt.Sprintf("bad metric identifier: %q contains %q", s, f))
			}
		}
	}
}

type Packet interface {
	AddValueList(ValueList) error
	Finalize() []byte
	Reset()
}

var ErrPacketFull = errors.New("not enough space")
