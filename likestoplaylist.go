// Copyright 2015 Mathieu Lonjaret

package main

import (
	//	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	//	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

const (
	idstring = "http://golang.org/pkg/http/#ListenAndServe"
	apiURL = "https://api.soundcloud.com"
)

var (
	flagHost         = flag.String("host", "0.0.0.0:8080", "listening port and hostname")
	help             = flag.Bool("h", false, "show this help")
	flagClientSecret = flag.String("client_secret", "", "app oauth2 client secret")
)

func usage() {
	fmt.Fprintf(os.Stderr, "\t likestoplaylist \n")
	flag.PrintDefaults()
	os.Exit(2)
}

var oauthToken string

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

type oauth2Response struct {
	Token string `json:"access_token,omitempty"`
	Scope string
}

func getToken(authCode string) (tok string, err error) {
	resp, err := http.PostForm("https://api.soundcloud.com/oauth2/token", url.Values{
		"client_id":     {"eba287aa2c6cef9b2cf5aed6b73d3542"},
		"client_secret": {*flagClientSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {"http://127.0.0.1:8080/callback"},
		"code":          {authCode},
	})
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var jsonResp oauth2Response
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return "", fmt.Errorf("could not decode %q: %v", body, err)
	}
	tok = jsonResp.Token
	if tok == "" {
		return "", fmt.Errorf("No token in response: %q", body)
	}
	return tok, nil
}

func serveError(w http.ResponseWriter, response string, err error) {
	log.Printf("%v", err)
	w.Write([]byte(response))
}

func apiGet(url string) ([]byte, error) {
	resp, err := http.Get(apiURL+url+"?oauth_token=" + oauthToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	authCode := r.FormValue("code")
	if authCode == "" {
		serveError(w, "no auth code in callback", fmt.Errorf("no auth code in callback"))
		return
	}
	tok, err := getToken(authCode)
	if err != nil {
		serveError(w, "token exchange error", fmt.Errorf("token exchange error: %v", err))
		return
	}
	println("GOT TOKEN: " + tok)
	oauthToken = tok

	body, err := apiGet("/me")
	if err != nil {
		serveError(w, "error getting me", fmt.Errorf("error getting me: %v", err))
		return
	}

	println(string(body))

	body, err = apiGet("/me/favorites")
	if err != nil {
		serveError(w, "error getting my favorites", fmt.Errorf("error getting my favorites: %v", err))
		return
	}

//	println(string(body))
	w.Write("ALL GOOD")
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *help {
		usage()
	}
	if *flagClientSecret == "" {
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
		handleCallback(w, r)
		return
	}))
	errc := make(chan error)
	go func() {
		errc <- http.ListenAndServe(*flagHost, nil)
	}()
	if err := exec.Command("xdg-open", "https://soundcloud.com/connect/?client_id=eba287aa2c6cef9b2cf5aed6b73d3542&redirect_uri=http://127.0.0.1:8080/callback&response_type=code&scope=non-expiring").Run(); err != nil {
		log.Fatal(err)
	}
	log.Fatal(<-errc)
}

/*
https://soundcloud.com/connect/?client_id=eba287aa2c6cef9b2cf5aed6b73d3542&redirect_uri=http://127.0.0.1:8080/callback&response_type=code&scope=non-expiring

code=52a31aaa7ef5dc1f4cee4ebe0eeaef92

curl -F 'client_id=eba287aa2c6cef9b2cf5aed6b73d3542' -F 'grant_type=authorization_code' -F 'redirect_uri=http://127.0.0.1:8080/callback' -F 'code=52a31aaa7ef5dc1f4cee4ebe0eeaef92' https://api.soundcloud.com/oauth2/token

*/
