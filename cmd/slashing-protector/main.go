package main

import (
	"log"
	"net/http"

	_ "net/http/pprof"

	"github.com/alecthomas/kong"
	protectorhttp "github.com/bloxapp/slashing-protector/http"
	"github.com/bloxapp/slashing-protector/protector"
	"go.uber.org/zap"
)

var CLI struct {
	DbPath string `env:"DB_PATH" description:"Path to the database directory" default:"/slashing-protector-data"`
	Addr   string `env:"ADDR" description:"HTTP address to serve slashing-protector on" default:":9369"`
	Pprof  string `env:"PPROF" description:"HTTP address to serve pprof on"`
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

	// Serve pprof over HTTP.
	if CLI.Pprof != "" {
		go func() {
			// Ignore: G114: Use of net/http serve function that has no support for setting timeouts
			/* #nosec */
			log.Println(http.ListenAndServe(CLI.Pprof, nil))
		}()
	}

	// Create the server and start it.
	prtc := protector.New(CLI.DbPath)
	srv := protectorhttp.NewServer(logger, prtc)
	err = http.ListenAndServe(CLI.Addr, srv)
	logger.Fatal("ListenAndServe", zap.Error(err))
}
