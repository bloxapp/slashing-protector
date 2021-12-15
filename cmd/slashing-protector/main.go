package main

import (
	"log"
	"net/http"

	httpserver "github.com/bloxapp/slashing-protector/http"
	"github.com/bloxapp/slashing-protector/protector"
)

func main() {
	prtc := protector.New("./test")
	srv := httpserver.New(prtc)
	log.Fatal(http.ListenAndServe(":8080", srv))
}
