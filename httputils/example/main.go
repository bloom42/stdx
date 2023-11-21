package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/bloom42/stdx/httputils"
)

//go:embed webapp/*
var webapp embed.FS

func main() {
	server := http.NewServeMux()
	webappFS, _ := fs.Sub(webapp, "webapp")
	webappHandler, err := httputils.WebappHandler(webappFS)
	if err != nil {
		log.Fatal(err)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server.HandleFunc("/", webappHandler)
	err = http.ListenAndServe(":"+port, server)
	if err != nil {
		log.Fatal(err)
	}
}
