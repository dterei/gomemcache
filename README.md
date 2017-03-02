# Golang Memcache

A simple Golang implementation of memcache.

## Building

```
$ make
```

You'll need Golang installed to build, any recent version (e.g., 1.6+) should
work.

## Testing

To run the full test-suite:

```
$ make testfull
```

This runs a set of unit tests (`make test`) and a set of end-to-end tests
(`make testclient`) that launches the server and runs a memcache client against
it. All tests run with Golang's race detector enabled.

## Usage

```
$ ./bin/memcache
```

The server has a hard-coded 100MB limit for stored key-value pairs. No
command-line parsing is implemented, you'll need to edit
`src/memcached/main.go` to change the port or memory limit.

## Protocol Coverage

We support a limited part of the memcache binary protocol:

* `get`
* `set`
* `delete`

We support the `CAS` (or version) field, but we don't support expiration. We
also support an LRU eviction policy with a configurable memory limit
constraint.

## Performance Expectations

We get a fair amount for free by using Go:

* Light-weight threads multiplexed over N kernel threads (where N is usually
  the number of cores you have).
* Multiplexing (aka two-level scheduling) done using non-blocking, event-based
  IO.

This gives us a fairly performant system that is still easy to program in due
to the perception of blocking calls. The main limitation is that contention
that will arise on the single Mutex used to protect the hashmap.

It's tempting to think we could use a RWMutex as a really easy improvement, but
`gets` also perform a write due to updating the shared LRU. Memcached improves
this situation by having multiple LRU's (one per slab-class) and using a
shareded-lock to protect the hashmap too. It also avoids updating the LRU on
every requests by recording when the key was last updated in the LRU and only
updating keys that have not been updated for at least 1 second.

## Performance Measurement

Measuring on two machines over a 10GbE network. We evaluate simply by measuring
how many request-per-second we can achieve with a 95th percentile latency of
under 1ms.

We make use [mutilate](https://github.com/dterei/mutilate) to evaluate the
performance.

For reference, here are the mainline memcached numbers:

* 100% read,  1 core:   152K req/s
* 100% read,  4 core:   565K req/s
* 100% read, 12 core: 1,015K req/s
*  95% read,  1 core:   146K req/s
*  95% read,  4 core:   518K req/s
*  95% read, 12 core:   841K req/s

The numbers for our Go implementation are:

* 100% read,  1 core:    98K req/s
* 100% read,  4 core:   337K req/s
* 100% read, 12 core:   435K req/s
*  95% read,  1 core:    96K req/s
*  95% read,  4 core:   331K req/s
*  95% read, 12 core:   426K req/s

So about half the performance when using Golang 1.8.

## Licensing

This library is BSD-licensed.

## Authors

This library is written and maintained by David Terei, <code@davidterei.com>.

