package collectd

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestWritePlainTextPacket(t *testing.T) {
	b := NewPlainTextPacket()

	m := Metric{
		Host:       "example.com",
		Plugin:     "golang",
		Type:       "gauge",
		ValueTypes: []ValueType{DERIVE},
		Interval:   10 * time.Second,
	}

	v := ValueList{
		Metric: &m,
		Time:   time.Unix(1426076671, 123000000), // Wed Mar 11 13:24:31 CET 2015
		Values: []float64{1},
	}

	if err := b.AddValueList(v); err != nil {
		t.Errorf("Write got %v, want nil", err)
		return
	}

	m.PluginInstance = "test"
	v.Time = time.Unix(1426076681, 234000000) // Wed Mar 11 13:24:41 CET 2015
	v.Values = []float64{2}

	if err := b.AddValueList(v); err != nil {
		t.Errorf("Write got %v, want nil", err)
		return
	}

	want := []byte{
		// vl1
		0, 0, 0, 16, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0,
		0, 2, 0, 11, 'g', 'o', 'l', 'a', 'n', 'g', 0,
		0, 4, 0, 10, 'g', 'a', 'u', 'g', 'e', 0,
		// 1426076671.123 * 2^30 = 1531238166015458148.352
		// 1531238166015458148 = 0x15400cffc7df3b64
		0, 8, 0, 12, 0x15, 0x40, 0x0c, 0xff, 0xc7, 0xdf, 0x3b, 0x64,
		0, 9, 0, 12, 0, 0, 0, 0x02, 0x80, 0, 0, 0,
		0, 6, 0, 15, 0, 1, 2, 0, 0, 0, 0, 0, 0, 0, 1,
		// vl2
		0, 3, 0, 9, 't', 'e', 's', 't', 0,
		//1426076681.234 * 2^30 = 1531238176872061730.816
		//1531238176872061731 = 0x15400d024ef9db23
		0, 8, 0, 12, 0x15, 0x40, 0x0d, 0x02, 0x4e, 0xf9, 0xdb, 0x23,
		0, 6, 0, 15, 0, 1, 2, 0, 0, 0, 0, 0, 0, 0, 2,
	}
	got := b.Finalize()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func BenchmarkFormatPlainText(bench *testing.B) {
	b := NewPlainTextPacket()
	bench.ReportAllocs()
	m := Metric{
		Host:       "example.com",
		Plugin:     "golang",
		Type:       "foobar",
		ValueTypes: []ValueType{DERIVE, GAUGE},
		Interval:   10 * time.Second,
	}
	t := time.Unix(1426076671, 123000000)
	err := b.AddValues(&m, t, 1, math.NaN())
	if err != nil {
		bench.Fatal(err)
	}
	bench.ResetTimer()
	for n := 0; n < bench.N; n++ {
		b.AddValues(&m, t, 1, math.NaN())
		b.Reset()
	}
}

func TestWriteTime(t *testing.T) {
	b := NewPlainTextPacket()
	b.writeTime(time.Unix(1426083986, 314000000)) // Wed Mar 11 15:26:26 CET 2015

	// 1426083986.314 * 2^30 = 1531246020641985396.736
	// 1531246020641985397 = 0x1540142494189375
	want := []byte{0, 8, // pkg type
		0, 12, // pkg len
		0x15, 0x40, 0x14, 0x24, 0x94, 0x18, 0x93, 0x75,
	}
	got := b.buffer.Bytes()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWriteValues(t *testing.T) {
	b := NewPlainTextPacket()

	v := ValueList{
		Metric: &Metric{
			ValueTypes: []ValueType{
				GAUGE, DERIVE, GAUGE,
			}},
		Values: []float64{
			42,
			31337,
			math.NaN(),
		},
	}

	b.writeValues(v)

	want := []byte{0, 6, // pkg type
		0, 33, // pkg len
		0, 3, // num values
		1, 2, 1, // gauge, derive, gauge
		0, 0, 0, 0, 0, 0, 0x45, 0x40, // 42.0
		0, 0, 0, 0, 0, 0, 0x7a, 0x69, // 31337
		0, 0, 0, 0, 0, 0, 0xf8, 0x7f, // NaN
	}
	got := b.buffer.Bytes()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWriteString(t *testing.T) {
	b := NewPlainTextPacket()

	if err := b.writeString(0xf007, "foo"); err != nil {
		t.Errorf("got %v, want nil", err)
	}

	want := []byte{0xf0, 0x07, // pkg type
		0, 8, // pkg len
		'f', 'o', 'o', 0, // "foo\0"
	}
	got := b.buffer.Bytes()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWriteInt(t *testing.T) {
	b := NewPlainTextPacket()

	if err := b.writeInt(23, uint64(384)); err != nil {
		t.Errorf("got %v, want nil", err)
	}

	want := []byte{0, 23, // pkg type
		0, 12, // pkg len
		0, 0, 0, 0, 0, 0, 1, 128, // 384
	}
	got := b.buffer.Bytes()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
