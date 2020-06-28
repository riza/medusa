# Medusa
> Fastest recursive HTTP fuzzer, like a Ferrari.

[![Travis](https://img.shields.io/travis/riza/medusa.svg)](https://travis-ci.org/riza/medusa)  [![GitHub version](https://badge.fury.io/gh/riza%2Fmedusa.svg)](https://github.com/riza/medusa/releases) [![Go Report Card](https://goreportcard.com/badge/github.com/riza/medusa)](https://goreportcard.com/report/github.com/riza/medusa) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/riza/medusa) [![codecov](https://codecov.io/gh/riza/medusa/branch/master/graph/badge.svg)](https://codecov.io/gh/riza/medusa)

![demo](https://github.com/riza/medusa/blob/master/res/demo.png?raw=true)


## Usage
```
Usage: medusa [options...]
Options:
-u                    Single URL  
-uL                   URL list file path (line by line)
-e                    Extension 
-s                    Force schema (uses default http if does not contains url)
-ua                   User-agent value (default %s)
-cP                   Postive status codes (seperate by comma)
-cN                   Negative status codes (seperate by comma)
-x                    Bypass SSL verification
-t                    HTTP response timeout (10s)
-r                    Enable recursive fuzzing
-w                    Directory wordlist (line by line)
-v                    Verbose mode, show logs
-conc                 Maximum concurrent requests
-cpus                 Number of used cpu cores.
```

## Benchmarks

```
TODO
```

## Known issues

<details><summary>socket: too many open file</summary>
<p>

#### The solution to this is to increase ulimit, you can solve this problem by typing `ulimit -n 8129` before running Medusa.

</p>
</details>
## Where does the name Medusa come from?
```
TODO
```

