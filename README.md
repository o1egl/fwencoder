# Fixed width file parser in GO (golang)
[![License](http://img.shields.io/:license-mit-blue.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/o1egl/fwparser?status.svg)](https://godoc.org/github.com/o1egl/fwparser)
[![Build Status](http://img.shields.io/travis/o1egl/fwparser.svg?style=flat-square)](https://travis-ci.org/o1egl/fwparser)
[![Coverage Status](http://img.shields.io/coveralls/o1egl/fwparser.svg?style=flat-square)](https://coveralls.io/r/o1egl/fwparser)
[![Go Report Card](https://goreportcard.com/badge/github.com/o1egl/fwparser)](https://goreportcard.com/report/github.com/o1egl/fwparser)

This library allows you to parse fixed width table data like:

```
Name            Address               Postcode Phone          Credit Limit Birthday
Evan Whitehouse V4560 Camel Back Road 3122     (918) 605-5383    1000000.5 19870101
Chuck Norris    P.O. Box 872          77868    (713) 868-6003     10909300 19651203
```

## Install

To install the library use the following command:

```
$ go get -u github.com/o1egl/fwparser
```

## Example

Parsing data from io.Reader:

```go
type Person struct {
	Name        string
	Address     string
	Postcode    int
	Phone       string
	CreditLimit float64   `json:"Credit Limit"`
	Bday        time.Time `column:"Birthday" format:"20060102"`
}

f, _ := os.Open("/path/to/file")
defer f.Close

var people []Person
fwparser.UnmarshalReader(f, &people)

// You can also parse data from byte array

b, err := ioutil.ReadFile("/path/to/file")
var people []Person
fwparser.UnmarshalReader(f, &people)
```

You can also parse data from byte array:

```go
b, _ := ioutil.ReadFile("/path/to/file")
var people []Person
fwparser.Unmarshal(b, &people)
```