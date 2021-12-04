# typemap

[![GoDoc](https://img.shields.io/badge/api-Godoc-blue.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/jxskiss/typemap)][goreport]
[![Issues](https://img.shields.io/github/issues/jxskiss/typemap.svg)][issues]
[![GitHub release](http://img.shields.io/github/release/jxskiss/typemap.svg)][release]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg)][license]

[godoc]: https://pkg.go.dev/github.com/jxskiss/typemap
[goreport]: https://goreportcard.com/report/github.com/jxskiss/typemap
[issues]: https://github.com/jxskiss/typemap/issues
[release]: https://github.com/jxskiss/typemap/releases
[license]: https://github.com/jxskiss/typemap/blob/master/LICENSE

Package typemap implements a fast concurrent safe map suitable to cache type related information.
It copies data when writing to the map (copy-on-write), thus it's lock-free and achieves
super high [performance](#performance) for concurrent reading operations.

The open addressing linear probing hash table is forked from [intintmap](https://github.com/brentp/intintmap).

Related articles:

1. [Implementing a world fastest Java int-to-int hash map](http://java-performance.info/implementing-world-fastest-java-int-to-int-hash-map/)
1. [Fibonacci Hashing: The Optimization that the World Forgot (or: a Better Alternative to Integer Modulo)](https://probablydance.com/2018/06/16/fibonacci-hashing-the-optimization-that-the-world-forgot-or-a-better-alternative-to-integer-modulo/)

## Usage

```go
typeCache := typemap.NewTypeMap()

typeCache.SetByType(reflect.TypeOf(someType{}), something) // or
typeCache.SetByUintptr(uintptr(typePointer), something)

typeCache.GetByType(reflect.TypeOf(someType{})) // or
typeCache.GetByUintptr(uintptr(typePointer))
```

## Performance

It is 14X faster than sync.Map, and 170X faster than the builtin map with RWMutex.

```text
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz

Benchmark_Concurrent_StdMap_Get_NoLock-12       79640444                13.70 ns/op
Benchmark_Concurrent_StdMap_Get_RWMutex-12       2537535               473.0 ns/op
Benchmark_Concurrent_SyncMap_Get-12             31426616                37.56 ns/op
Benchmark_Concurrent_Slice_Index-12           1000000000                 0.6455 ns/op
Benchmark_Concurrent_TypeMap_Get-12            415275356                 2.761 ns/op
```

Some notes to tune performance:

```shell
# check inline cost information
go build -gcflags=-m=2 ./

# check bounds check elimination information
go build -gcflags="-d=ssa/check_bce/debug=1" ./

# check assembly output
go tool compile -S ./intintmap.go
```
