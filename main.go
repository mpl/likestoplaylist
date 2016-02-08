// Copyright 2015 Mathieu Lonjaret

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

const (
	idstring = "http://golang.org/pkg/http/#ListenAndServe"
)

var (
	host = flag.String("host", "0.0.0.0:8080", "listening port and hostname")
	help = flag.Bool("h", false, "show this help")
)

func usage() {
	fmt.Fprintf(os.Stderr, "\t likestoplaylist \n")
	flag.PrintDefaults()
	os.Exit(2)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
		}()
		title := r.URL.Path
		w.Header().Set("Server", idstring)
		fn(w, r, title)
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *help {
		usage()
	}

	nargs := flag.NArg()
	if nargs > 0 {
		usage()
	}

	http.Handle("/", makeHandler(func(w http.ResponseWriter, r *http.Request, title string) {
		println(title)
		w.Write([]byte("OK ALRIGHT"))
		return
	}))
	http.Handle("/callback", makeHandler(func(w http.ResponseWriter, r *http.Request, title string) {
		println(title)
		w.Write([]byte(r.URL.RawQuery))
		return
	}))
	http.ListenAndServe(*host, nil)
}

/*
https://soundcloud.com/connect/?client_id=eba287aa2c6cef9b2cf5aed6b73d3542&redirect_uri=urn:ietf:wg:oauth:2.0:oob&scope=*
https://soundcloud.com/connect/?client_id=eba287aa2c6cef9b2cf5aed6b73d3542&redirect_uri=http://127.0.0.1:8080/callback&response_type=code&scope='*'

response_type	enumeration	(code, token)
scope	string	'*'
display	string	Can specify a value of 'popup' for mobile optimized screen
state	string	Any value included here will be appended to the redirect URI

*/
