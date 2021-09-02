package cdclient

import (
	"math"
	"testing"
	"time"
)

func TestExecFormatter(t *testing.T) {
	ef := ExecFormatter{}

	m := Metric{
		Host:           "example.com",
		Plugin:         "golang",
		PluginInstance: "foo",
		Type:           "gauge",
		TypeInstance:   "bar",
		DSTypes:        []DSType{DERIVE, COUNTER},
		Interval:       10 * time.Second,
	}

	v := ValueList{
		Metric: &m,
		Time:   time.Unix(1426076671, 123000000), // Wed Mar 11 13:24:31 CET 2015
		Values: []float64{1, 12.3},
	}

	_ = ef.AddValueList(v)

	got := string(ef.Finalize())
	expected := "putval example.com/golang-foo/gauge-bar interval=10 1426076671:1:12.3\n"
	if got != expected {
		t.Fatalf("%q != (expected)%q", got, expected)
	}
}

func BenchmarkExecFormatter(bench *testing.B) {
	ef := ExecFormatter{}
	bench.ReportAllocs()
	m := Metric{
		Host:     "example.com",
		Plugin:   "golang",
		Type:     "gauge",
		DSTypes:  []DSType{DERIVE, COUNTER},
		Interval: 10 * time.Second,
	}
	t := time.Unix(1426076671, 123000000)
	_ = ef.AddValues(&m, t, 1, math.NaN())
	ef.Finalize()
	ef.Reset()
	bench.ResetTimer()
	for n := 0; n < bench.N; n++ {
		ef.AddValues(&m, t, 1, math.NaN())
		ef.Finalize()
		ef.Reset()
	}
}
