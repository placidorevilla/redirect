package main

import (
	"flag"
	"net"
	"net/http"
	"time"
)

func main() {
	uiFolder := flag.String("ui", "./ui", "Location of UI files")
	uiAddr := flag.String("ui-addr", "127.0.0.1:10101", "Address for UI")
	configFile := flag.String("config", "./redir.json", "File to save configs")
	bind := flag.String("bind", "0.0.0.0:10100", "Redirect address")
	flag.Parse()

	mgr := newManager()
	_, port, err := net.SplitHostPort(*bind)
	if err != nil {
		panic(err)
	}

	uiMux := http.NewServeMux()
	uiServer := &http.Server{
		Addr:           *uiAddr,
		Handler:        uiMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	apiHandler := &api{mgr, port, *configFile}
	apiHandler.load()
	uiMux.Handle("/api/", http.StripPrefix("/api/", apiHandler))
	uiMux.Handle("/", http.FileServer(http.Dir(*uiFolder)))
	go func() { panic(uiServer.ListenAndServe()) }()

	mainServer := &http.Server{
		Addr:           *bind,
		Handler:        mgr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	panic(mainServer.ListenAndServe())
}
