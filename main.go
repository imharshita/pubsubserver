package main

import (
	"log"
	"net/http"

	pubsubserver "github.com/imharshita/pubsubserver/pubsub"
)

func main() {
	pubsub := pubsubserver.NewPubSubServer()

	s := &http.Server{
		Handler: pubsub,
		Addr:    ":8000",
	}

	log.Printf("listening on http://localhost:%v", s.Addr)
	log.Fatal(s.ListenAndServe())
}
