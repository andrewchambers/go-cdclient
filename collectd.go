// A fast package for sending metrics to collectd.
package cdclient

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type DSType byte

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
	DSTypes        []DSType
	Interval       time.Duration
}

type ValueList struct {
	Metric *Metric
	Time   time.Time
	Values []float64
}

func (m *Metric) Validate() {
	if m.Interval == 0 {
		panic("bad metric: interval is zero")
	}
	if m.Host == "" {
		panic("bad metric: host is empty")
	}
	if m.Plugin == "" {
		panic("bad metric: plugin is empty")
	}
	for _, v := range m.DSTypes {
		if v < 0 || v > ABSOLUTE {
			panic("bad metric: value type is out of range")
		}
	}
	for i, s := range []string{
		m.Host,
		m.Plugin,
		m.PluginInstance,
		m.Type,
		m.TypeInstance,
	} {
		if len(s) > 63 {
			panic(fmt.Sprintf("bad metric: %q is too long", s))
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
				panic(fmt.Sprintf("bad metric: mandatory field empty"))
			}
		}
		for _, f := range forbidden {
			if strings.IndexRune(s, f) != -1 {
				panic(fmt.Sprintf("bad metric: identifier %q contains %q", s, f))
			}
		}
	}
}

type MetricSink interface {
	AddValues(*Metric, time.Time, ...float64) error
	AddValueList(ValueList) error
}

type Packet interface {
	MetricSink
	Finalize() []byte
	Reset()
}

var ErrPacketFull = errors.New("not enough space")
