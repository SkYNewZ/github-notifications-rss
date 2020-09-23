package main

import (
	"log"
	"net/http"
	"time"

	f "github.com/SkYNewZ/github-notifications-rss/function"
)

func main() {
	var addr string = "127.0.0.1:8080"
	s := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	http.HandleFunc("/feed", f.GetGithubNotificationsJSONFeed)

	log.Printf("Listening on http://%s/feed\n", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
