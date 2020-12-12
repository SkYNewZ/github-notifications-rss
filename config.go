package main

import (
	"net"
	"net/url"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

var port = "8080"
var addr = "127.0.0.1"
var feedURL string
var useCache = true

func init() {
	log.SetLevel(log.DebugLevel)
	addr = validIP()
	feedURL = validURL()
	port = validPort()
	useCache = os.Getenv("NO_CACHE") != "1"
	if !useCache {
		log.Debugf("caching response is disabled")
	}
}

func validURL() string {
	if v := os.Getenv("FEED_URL"); v == "" {
		log.Fatalln("Missing $FEED_URL")
	}

	_, err := url.Parse(os.Getenv("FEED_URL"))
	if err != nil {
		log.Fatalln("Invalid $FEED_URL")
	}

	return os.Getenv("FEED_URL")
}

func validIP() string {
	if v := os.Getenv("LISTEN_ADDR"); v == "" {
		// ADDR not filled, return default
		return addr
	}

	if net.ParseIP(os.Getenv("LISTEN_ADDR")) == nil {
		log.Fatalln("Invalid $LISTEN_ADDR")
	}

	return os.Getenv("LISTEN_ADDR")
}

func validPort() string {
	if v := os.Getenv("PORT"); v == "" {
		// PORT not filled, return default
		return port
	}

	if _, err := strconv.Atoi(os.Getenv("PORT")); err != nil {
		log.Fatalln("Invalid $PORT")
	}

	return os.Getenv("PORT")
}
