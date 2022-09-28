package main

import (
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"
)

var feedURL string

const (
	defaultPort    string = "8080"
	defaultFeedURL string = "http://localhost:8080/feed"

	envFeedURL      string = "FEED_URL"
	envPort         string = "PORT"
	envDisableCache string = "NO_CACHE"
)

func init() {
	log.SetLevel(log.DebugLevel)
	feedURL = validURL()

	if CacheDisabled() {
		log.Debugf("caching response is disabled")
	}
}

func validURL() string {
	v := os.Getenv(envFeedURL)
	if v == "" {
		return defaultFeedURL
	}

	if _, err := url.Parse(v); err != nil {
		log.Fatalln("Invalid $FEED_URL provided")
	}

	return v
}

// CacheDisabled return true if cache is disabled.
func CacheDisabled() bool {
	return os.Getenv(envDisableCache) == "1"
}
