package main

import (
	"log"
	"net/http"

	httpserver "github.com/bloxapp/slashing-protector/http"
	"github.com/bloxapp/slashing-protector/protector"
)

func main() {
	log.Printf("Starting...")
	prtc := protector.New("/app/tmp")
	srv := httpserver.NewServer(prtc)
	log.Printf("Listening on :9369")
	log.Fatal(http.ListenAndServe(":9369", srv))
}
