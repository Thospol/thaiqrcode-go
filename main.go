package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"thaiqr-go/internal/handler"
)

func main() {
	port := flag.String("p", "8031", "port number")

	ro := handler.Routes{}
	hdlr := ro.InitRoute()
	srv := &http.Server{
		Addr:    fmt.Sprint(":", *port),
		Handler: hdlr,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Panicf("listen: %s\n", err)
	}
}
