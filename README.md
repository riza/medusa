<img align="right" width="200" src="https://github.com/riza/medusa/blob/master/res/logo_tmp.png?raw=true" />

# medusa [WIP]

[![Travis](https://img.shields.io/travis/riza/medusa.svg)](https://travis-ci.org/riza/medusa)
[![Go Report Card](https://goreportcard.com/badge/github.com/riza/medusa)](https://goreportcard.com/report/github.com/riza/medusa)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/riza/medusa)
[![codecov](https://codecov.io/gh/riza/medusa/branch/master/graph/badge.svg)](https://codecov.io/gh/riza/medusa)
[![GitHub version](https://badge.fury.io/gh/riza%2Fmedusa.svg)](https://github.com/riza/medusa/releases)

Blazingly fast directory fuzzer for HTTP

## Build

```bash
$ go build -tags netgo -installsuffix netgo
```

## Usage

```bash
Usage of ./medusa:
  -D string
        Multiple dir from file
  -H string
        Multiple host from file
  -boundary int
        Concurrent boundary limit (default 1024)
  -cc string
        Collect status code (--c 403,404) (default "200")
  -cpu int
        CPU Procs
  -d string
        Singular dir
  -depth int
        Depth level
  -h string
        Singular host
  -o string
        Output filename (default "output.txt")
  -retry int
        Retry limit per host (default 3)
  -timeout duration
        HTTP response timeout (default 500ns)
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
Please make sure to update tests as appropriate.

## TODO

- [ ] Test conditions
- [ ] ...


## License
[MIT](https://choosealicense.com/licenses/mit/)