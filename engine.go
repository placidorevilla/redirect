package redirect

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"text/template"
)

type engine struct {
	storage Storage
	stat    StatWriter
	lock    sync.RWMutex
	rules   map[string]*template.Template
}

// Create default engine based on provided storage and sink
func DefaultEngine(storage Storage, sink StatWriter) Engine {
	if storage == nil {
		panic("storage is nil")
	}
	if sink == nil {
		panic("stats sink is nil")
	}
	return &engine{
		storage: storage,
		stat:    sink,
	}
}

func (eng *engine) ServeHTTP(wr http.ResponseWriter, rq *http.Request) {
	defer rq.Body.Close()

	service := strings.Trim(rq.URL.Path, "/")

	// try to find redirect rule
	eng.lock.RLock()
	tpl, ok := eng.rules[service]
	eng.lock.RUnlock()
	if !ok {
		http.NotFound(wr, rq)
		return
	}
	// notify stat counter
	eng.stat.Touch(service)

	// render redirect template
	urlData := &bytes.Buffer{}
	err := tpl.Execute(urlData, rq)
	if err != nil {
		log.Println("engine: failed execute template for service", service, ":", err)
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}
	url := strings.TrimSpace(urlData.String())

	// We send TARGET in Location header on HEAD request with 200 OK status
	if rq.Method == "HEAD" {
		wr.Header().Add("Location", url)
		wr.WriteHeader(http.StatusOK)
		return
	}
	wr.Header().Add("Content-Length", "0")
	http.Redirect(wr, rq, url, http.StatusFound)
}

func (eng *engine) Reload() error {
	rules, err := eng.storage.All()
	if err != nil {
		return fmt.Errorf("engine: read rules from storage: %v", err)
	}
	var swap = make(map[string]*template.Template)
	for _, rule := range rules {
		t, err := template.New("").Parse(rule.LocationTemplate)
		if err != nil {
			return fmt.Errorf("engine: parse rule for url %v: %v", rule.URL, err)
		}
		swap[rule.URL] = t
	}
	eng.lock.Lock()
	eng.rules = swap
	eng.lock.Unlock()
	return nil
}
