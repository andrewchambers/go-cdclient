# go-fast-collectd

[![godocs.io](http://godocs.io/github.com/andrewchambers/go-fast-collectd?status.svg)](http://godocs.io/github.com/andrewchambers/go-fast-collectd)

A fast, dependency free implementation of the [collectd](https://github.com/collectd) binary network protocol.

## How fast?

This package can form an encrypted metric packet in less than 1 microsecond, and we can then publish it
with a single syscall.

This package has also been optimized such that there are no memory allocations in any of the public facing apis when used in a loop, so adds negligible garbage collection overhead to your program.

### Benchmark

go-fast-collectd 

```
BenchmarkFormatEncryptedUdpPacket-32   1343607   904.0 ns/op   0 B/op   0 allocs/op
```

Compared to the [official collectd go library](https://github.com/collectd/go-collectd/blob/master/network/buffer.go):

```
BenchmarkFormatEncryptedUdpPacket-32   244285   4643 ns/op   1736 B/op   37 allocs/op
```

We have more than a 4x speedup at writing encrypted metrics at the time of writing.

[benchmark source](https://gist.github.com/andrewchambers/5e50a90b904e8b23d73f613ca82911fe)

## Example

See the example directory for basic usage.

You should probably design a higher level api to use this package in an application.