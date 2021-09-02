package cdclient

import (
	"bytes"
	"strconv"
	"time"
)

type ExecFormatter struct {
	buffer     bytes.Buffer
	tmpBuf     []byte
	tmpValues  []float64
	NoInterval bool
}

func (ef *ExecFormatter) AddValues(m *Metric, t time.Time, values ...float64) error {
	// This copy allows the go compiler to avoid an allocation.
	ef.tmpValues = append(ef.tmpValues[:0], values...)
	return ef.AddValueList(ValueList{
		Metric: m,
		Time:   t,
		Values: ef.tmpValues,
	})
}

func (ef *ExecFormatter) AddValueList(vl ValueList) error {
	m := vl.Metric
	ef.buffer.WriteString("putval ")
	ef.buffer.WriteString(m.Host)
	ef.buffer.WriteByte('/')
	ef.buffer.WriteString(m.Plugin)
	if m.PluginInstance != "" {
		ef.buffer.WriteByte('-')
		ef.buffer.WriteString(m.PluginInstance)
	}
	ef.buffer.WriteByte('/')
	ef.buffer.WriteString(m.Type)
	if m.TypeInstance != "" {
		ef.buffer.WriteByte('-')
		ef.buffer.WriteString(m.TypeInstance)
	}
	ef.buffer.WriteByte(' ')
	if !ef.NoInterval {
		ef.buffer.WriteString("interval=")
		ef.tmpBuf = strconv.AppendInt(ef.tmpBuf[:0], int64(m.Interval.Seconds()), 10)
		ef.buffer.Write(ef.tmpBuf)
		ef.buffer.WriteByte(' ')
	}

	ef.tmpBuf = strconv.AppendInt(ef.tmpBuf[:0], vl.Time.Unix(), 10)
	ef.buffer.Write(ef.tmpBuf)
	for _, v := range vl.Values {
		ef.buffer.WriteByte(':')
		ef.tmpBuf = strconv.AppendFloat(ef.tmpBuf[:0], v, 'f', -1, 64)
		ef.buffer.Write(ef.tmpBuf)
	}

	ef.buffer.WriteByte('\n')
	return nil
}

func (ef *ExecFormatter) Finalize() []byte {
	return ef.buffer.Bytes()
}

func (ef *ExecFormatter) Reset() {
	ef.buffer.Reset()
	ef.tmpBuf = ef.tmpBuf[:0]
}
