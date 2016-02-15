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

type track struct {
	Kind string
	Id int64
}

func getFavorites() ([]track, error) {
	tracks := make([]track, 1)
	body, err := apiGet("/me/favorites")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &tracks); err != nil {
		return nil, fmt.Errorf("could not decode %q: %v", body, err)
	}
	fmt.Printf("found %d tracks in favorites\n", len(tracks))
	for _,v := range tracks {
		if v.Kind != "track" {
			// TODO(mpl): remove from list? error out?
			fmt.Printf("but %#v is actually not a track\n", v)
		}
	}
	return tracks, nil
}

/*
type playlist struct {
	Id int64
	UserId int
}

{
  "kind": "playlist",
  "id": 405726,
  "created_at": "2010/11/02 09:24:50 +0000",
  "user_id": 3207,
  "duration": 154516,
  "sharing": "public",
  "tag_list": "",
  "permalink": "field-recordings",
  "track_count": 5,
  "streamable": true,
  "downloadable": true,
  "embeddable_by": "me",
  "purchase_url": null,
  "label_id": null,
  "type": "other",
  "playlist_type": "other",
  "ean": "",
  "description": "a couple of field recordings to test http://soundiverse.com.\r\n\r\nrecorded with the fire recorder: http://soundcloud.com/apps/fire",
  "genre": "",
  "release": "",
  "purchase_title": null,
  "label_name": "",
  "title": "Field Recordings",
  "release_year": null,
  "release_month": null,
  "release_day": null,
  "license": "all-rights-reserved",
  "uri": "http://api.soundcloud.com/playlists/405726",
  "permalink_url": "http://soundcloud.com/jwagener/sets/field-recordings",
  "artwork_url": "http://i1.sndcdn.com/artworks-000025801802-1msl1i-large.jpg?5e64f12",
  "user": {
    "id": 3207,
    "kind": "user",
    "permalink": "jwagener",
    "username": "Johannes Wagener",
    "uri": "http://api.soundcloud.com/users/3207",
    "permalink_url": "http://soundcloud.com/jwagener",
    "avatar_url": "http://i1.sndcdn.com/avatars-000014428549-3at7qc-large.jpg?5e64f12"
  },
  "tracks": [
    {
      "kind": "track",
      "id": 6621631,
      "created_at": "2010/11/02 09:08:43 +0000",
      "user_id": 3207,
      "duration": 27099,
      "commentable": true,
      "state": "finished",
      "original_content_size": 2382624,
      "sharing": "public",
      "tag_list": "Fieldrecording geo:lat=52.527544 geo:lon=13.402905",
      "permalink": "coffee-machine",
      "streamable": true,
      "embeddable_by": "all",
      "downloadable": false,
      "purchase_url": null,
      "label_id": null,
      "purchase_title": null,
      "genre": "",
      "title": "coffee machine",
      "description": "",
      "label_name": "",
      "release": "",
      "track_type": "",
      "key_signature": "",
      "isrc": "",
      "video_url": null,
      "bpm": null,
      "release_year": null,
      "release_month": null,
      "release_day": null,
      "original_format": "wav",

func createNewPlaylist(likes []track) error {

}
*/

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

	if _, err := getFavorites(); err != nil {
		serveError(w, "error getting favorites", fmt.Errorf("error getting favorites: %v", err))
		return
	}

	w.Write([]byte("ALL GOOD"))
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
