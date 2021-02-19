package main

import (
	_ "embed"
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/reddec/redirect"
)

func main() {
	uiFolder := flag.String("ui", "", "Location of custom UI files")
	uiAddr := flag.String("ui-addr", "127.0.0.1:10101", "Address for UI")
	configFile := flag.String("config", "./redir.json", "File to save configs")
	bind := flag.String("bind", "0.0.0.0:10100", "Redirect address")
	flag.Parse()

	// get redirect port for UI
	_, port, _ := net.SplitHostPort(*bind)

	// init defaults
	stats := redirect.InMemoryStats()
	storage := &redirect.JsonStorage{FileName: *configFile}
	engine := redirect.DefaultEngine(storage, stats)
	ui := redirect.DefaultUI(storage, stats, engine, port)

	go func() {
		panic(http.ListenAndServe(*bind, engine))
	}()

	static := http.FileServer(http.FS(redirect.DefaultUIStatic))
	if *uiFolder != "" {
		static = http.FileServer(http.Dir(*uiFolder))
	}
	http.Handle("/ui/", static)
	http.Handle("/api/", http.StripPrefix("/api/", ui))
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		// redirect to ui
		http.Redirect(writer, request, "ui/", http.StatusTemporaryRedirect)
	})
	fmt.Println("UI:", *uiAddr)
	fmt.Println("Bind:", *bind)
	panic(http.ListenAndServe(*uiAddr, nil))
}
