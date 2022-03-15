package main

import (
	"log"
	"net/http"

	"github.com/alecthomas/kong"
	protectorhttp "github.com/bloxapp/slashing-protector/http"
	"github.com/bloxapp/slashing-protector/protector"
	"go.uber.org/zap"
)

var CLI struct {
	DbPath string `env:"DB_PATH" description:"Path to the database directory" default:"/slashing-protector-data"`
	Addr   string `env:"ADDR" description:"Address to listen on" default:":9369"`
}

func main() {
	kong.Parse(&CLI)

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	// Display the configuration. Don't expose sensitive attributes!
	logger.Debug("Starting slashing-protector",
		zap.String("db_path", CLI.DbPath),
		zap.String("addr", CLI.Addr),
	)

	// Create the server and start it.
	prtc := protector.New(CLI.DbPath)
	srv := protectorhttp.NewServer(logger, prtc)
	err = http.ListenAndServe(CLI.Addr, srv)
	logger.Fatal("ListenAndServe", zap.Error(err))
}
