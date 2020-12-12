package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/feed", getGithubNotificationsJSONFeed)
	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "OK")
	})

	// Log global router access logs
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)

	srv := &http.Server{
		Addr:         addr + ":" + port,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      loggedRouter,
	}

	go func() {
		log.Printf("Listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}
