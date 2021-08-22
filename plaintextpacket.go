package collectd

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"
)

// DefaultBufferSize is the default size of "Buffer". This is based on the
// maximum bytes that fit into an Ethernet frame without fragmentation:
//   <Ethernet frame> - (<IPv6 header> + <UDP header>) = 1500 - (40 + 8) = 1452
const DefaultBufferSize = 1452
const MinimumBufferSize = 256

// IDs of the various "parts", i.e. subcomponents of a packet.
const (
	typeHost           = 0x0000
	typeTime           = 0x0001
	typeTimeHR         = 0x0008
	typePlugin         = 0x0002
	typePluginInstance = 0x0003
	typeType           = 0x0004
	typeTypeInstance   = 0x0005
	typeValues         = 0x0006
	typeInterval       = 0x0007
	typeIntervalHR     = 0x0009
	typeSignSHA256     = 0x0200
	typeEncryptAES256  = 0x0210
)

type PlainTextPacket struct {
	buffer    bytes.Buffer
	tmpValues []float64
	size      int
	// Encoding is stateful, these are last seen values.
	stateHost           string
	statePlugin         string
	statePluginInstance string
	stateInterval       time.Duration
	stateType           string
	stateTypeInstance   string
	stateTime           time.Time
}

func NewPlainTextPacket() *PlainTextPacket {
	return NewPlainTextPacketSize(DefaultBufferSize)
}

// NewBuffer initializes a new metric buffer,
// panics if the size is smaller than MinimumBufferSize.
func NewPlainTextPacketSize(size int) *PlainTextPacket {
	b := &PlainTextPacket{}
	b.init(size)
	return b
}

func (b *PlainTextPacket) init(size int) {
	if size < MinimumBufferSize {
		panic("buffer size to small")
	}
	b.size = size
	b.buffer.Grow(size)
	b.Reset()
}

func (b *PlainTextPacket) available() int {
	if b.size < b.buffer.Len() {
		return 0
	}
	return b.size - b.buffer.Len()
}

func (b *PlainTextPacket) AddValues(m *Metric, t time.Time, values ...float64) error {
	// This copy allows the go compiler to avoid an allocation.
	b.tmpValues = append(b.tmpValues[:0], values...)
	return b.AddValueList(ValueList{
		Metric: m,
		Time:   t,
		Values: b.tmpValues,
	})
}

func (b *PlainTextPacket) AddValueList(v ValueList) error {
	l := b.buffer.Len()
	if err := b.addValueList(v); err != nil {
		b.buffer.Truncate(l)
		return err
	}
	return nil
}

func (b *PlainTextPacket) addValueList(v ValueList) error {
	if err := b.writeIdentifier(v.Metric); err != nil {
		return err
	}
	if err := b.writeTime(v.Time); err != nil {
		return err
	}
	if err := b.writeInterval(v.Metric.Interval); err != nil {
		return err
	}
	if err := b.writeValues(v); err != nil {
		return err
	}
	return nil
}

func (b *PlainTextPacket) writeIdentifier(m *Metric) error {
	if m.Host != b.stateHost {
		if err := b.writeString(typeHost, m.Host); err != nil {
			return err
		}
		b.stateHost = m.Host
	}
	if m.Plugin != b.statePlugin {
		if err := b.writeString(typePlugin, m.Plugin); err != nil {
			return err
		}
		b.statePlugin = m.Plugin
	}
	if m.PluginInstance != b.statePluginInstance {
		if err := b.writeString(typePluginInstance, m.PluginInstance); err != nil {
			return err
		}
		b.statePluginInstance = m.PluginInstance
	}
	if m.Type != b.stateType {
		if err := b.writeString(typeType, m.Type); err != nil {
			return err
		}
		b.stateType = m.Type
	}
	if m.TypeInstance != b.stateTypeInstance {
		if err := b.writeString(typeTypeInstance, m.TypeInstance); err != nil {
			return err
		}
		b.stateTypeInstance = m.TypeInstance
	}
	return nil
}

func cdtimeFromNano(ns uint64) uint64 {
	s := (ns / 1000000000) << 30
	ns = (ns % 1000000000) << 30
	ns = (ns + 500000000) / 1000000000
	return s | ns
}

func (b *PlainTextPacket) writeTime(t time.Time) error {
	if b.stateTime.Equal(t) {
		return nil
	}
	b.stateTime = t
	return b.writeInt(typeTimeHR, cdtimeFromNano(uint64(t.UnixNano())))
}

func (b *PlainTextPacket) writeInterval(d time.Duration) error {
	if b.stateInterval == d {
		return nil
	}
	b.stateInterval = d
	return b.writeInt(typeIntervalHR, cdtimeFromNano(uint64(d.Nanoseconds())))
}

func (b *PlainTextPacket) writeValues(v ValueList) error {
	m := v.Metric
	values := v.Values
	size := 6 + 9*len(values)
	if size > b.available() {
		return ErrPacketFull
	}
	tmp := [8]byte{}
	binary.BigEndian.PutUint16(tmp[0:2], uint16(typeValues))
	binary.BigEndian.PutUint16(tmp[2:4], uint16(size))
	binary.BigEndian.PutUint16(tmp[4:6], uint16(len(values)))
	b.buffer.Write(tmp[:6])

	for _, t := range m.ValueTypes {
		b.buffer.WriteByte(uint8(t))
	}
	for i, v := range values {
		t := m.ValueTypes[i]
		switch t {
		case GAUGE:
			if math.IsNaN(float64(v)) {
				tmp = [8]byte{0, 0, 0, 0, 0, 0, 0xf8, 0x7f}
			} else {
				binary.LittleEndian.PutUint64(tmp[:], math.Float64bits(v))
			}
		case DERIVE:
			binary.BigEndian.PutUint64(tmp[:], uint64(int64(v)))
		case ABSOLUTE:
			fallthrough
		case COUNTER:
			binary.BigEndian.PutUint64(tmp[:], uint64(v))
		default:
			panic("unknown type")
		}
		b.buffer.Write(tmp[:])
	}
	return nil
}

func (b *PlainTextPacket) writeString(typ uint16, s string) error {
	size := len(s) + 4 + 1
	if size > b.available() {
		return ErrPacketFull
	}
	tmp := [4]byte{}
	binary.BigEndian.PutUint16(tmp[0:2], typ)
	binary.BigEndian.PutUint16(tmp[2:], uint16(size))
	b.buffer.Write(tmp[:])
	b.buffer.WriteString(s)
	b.buffer.WriteByte(0)
	return nil
}

func (b *PlainTextPacket) writeInt(typ uint16, n uint64) error {
	size := 12
	if size > b.available() {
		return ErrPacketFull
	}
	tmp := [12]byte{}
	binary.BigEndian.PutUint16(tmp[0:2], typ)
	binary.BigEndian.PutUint16(tmp[2:4], uint16(size))
	binary.BigEndian.PutUint64(tmp[4:], n)
	b.buffer.Write(tmp[:])
	return nil
}

func (b *PlainTextPacket) Finalize() []byte {
	return b.buffer.Bytes()
}

func (b *PlainTextPacket) Reset() {
	b.buffer.Reset()
	b.stateHost = ""
	b.statePlugin = ""
	b.statePluginInstance = ""
	b.stateInterval = 0
	b.stateType = ""
	b.stateTypeInstance = ""
	b.stateTime = time.Time{}
}
