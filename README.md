# Fixed width file parser (encoder/decoder) for GO (golang)
[![License](http://img.shields.io/:license-mit-blue.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/o1egl/fwencoder?status.svg)](https://godoc.org/github.com/o1egl/fwencoder)
![Build Status](https://github.com/o1egl/fwencoder/actions/workflows/build.yml/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/o1egl/fwencoder/branch/master/graph/badge.svg?token=BPBYoYAeZ0)](https://codecov.io/gh/o1egl/fwencoder)
[![Go Report Card](https://goreportcard.com/badge/github.com/o1egl/fwencoder)](https://goreportcard.com/report/github.com/o1egl/fwencoder)

This library is using to parse fixed-width table data like:

```
Name            Address               Postcode Phone          Credit Limit Birthday
Evan Whitehouse V4560 Camel Back Road 3122     (918) 605-5383    1000000.5 19870101
Chuck Norris    P.O. Box 872          77868    (713) 868-6003     10909300 19651203
```

## Install

To install the library use the following command:

```
$ go get -u github.com/o1egl/fwencoder
```

## Decoding example

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
err := fwencoder.UnmarshalReader(f, &people)
```

You can also parse data from byte array:

```go
b, _ := ioutil.ReadFile("/path/to/file")
var people []Person
err := fwencoder.Unmarshal(b, &people)
```


## Encoding example

```go
people := []Person{
	Name: "John",
	Address: "P.O. Box 872",
	Phone: "(713) 868-6003", 
	CreditLimit: 10909300,
	Bday: "19651203"
}

b, err := Marshal(&people)
fmt.Println(string(b))
```

or you can directly write to io.Writer

```go
people := []Person{
	Name: "John",
	Address: "P.O. Box 872",
	Phone: "(713) 868-6003", 
	CreditLimit: 10909300,
	Bday: "19651203"
}

err := MarshalWriter(os.Stdout, &people)
```
