package redirect

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	formFieldTemplate = "template"
	formFieldService  = "service"
	headerRedirPort   = "X-Redir-Port"
)

//go:embed ui/*
var DefaultUIStatic embed.FS

// description of rule for API request
type UIEntry struct {
	Template string `json:"template"`
	Hits     int64  `json:"hits"`
	URL      string `json:"url"`
}

type basicUI struct {
	storage   Storage
	stats     StatReader
	engine    Engine
	redirPort string
}

func DefaultUI(storage Storage, stats StatReader, engine Engine, redirPort string) http.Handler {
	if storage == nil {
		panic("ui storage is nil")
	}
	if stats == nil {
		panic("ui stats reader is nil")
	}
	if engine == nil {
		panic("ui engine ref is nil")
	}
	return &basicUI{
		stats:     stats,
		storage:   storage,
		engine:    engine,
		redirPort: redirPort,
	}
}

func (ui *basicUI) ServeHTTP(wr http.ResponseWriter, rq *http.Request) {
	defer rq.Body.Close()
	service := strings.Trim(rq.URL.Path, "/")
	switch rq.Method {
	case http.MethodGet:
		if service == "" {
			ui.list(wr, rq)
		} else {
			ui.get(service, wr, rq)
		}
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		ui.set(service, wr, rq)
	case http.MethodDelete:
		ui.remove(service, wr, rq)
	default:
		ui.list(wr, rq)
	}
}

func (ui *basicUI) list(wr http.ResponseWriter, rq *http.Request) {
	var ans = make(map[string]*UIEntry)
	entries, err := ui.storage.All()
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, elem := range entries {
		ans[elem.URL] = &UIEntry{
			URL:      elem.URL,
			Template: elem.LocationTemplate,
			Hits:     ui.stats.Visits(elem.URL),
		}
	}
	wr.Header().Set(headerRedirPort, ui.redirPort)
	sendJSON(ans, wr)
}

func (ui *basicUI) get(service string, wr http.ResponseWriter, rq *http.Request) {
	template, exists := ui.storage.Get(service)
	if !exists {
		http.NotFound(wr, rq)
		return
	}
	wr.Header().Set(headerRedirPort, ui.redirPort)
	sendJSON(&UIEntry{
		URL:      service,
		Hits:     ui.stats.Visits(service),
		Template: template,
	}, wr)
}

func (ui *basicUI) remove(service string, wr http.ResponseWriter, rq *http.Request) {
	err := ui.storage.Remove(service)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}
	err = ui.engine.Reload()
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}
	wr.WriteHeader(http.StatusNoContent)
}

func (ui *basicUI) set(service string, wr http.ResponseWriter, rq *http.Request) {
	var entry UIEntry
	if strings.Contains(rq.Header.Get("Content-Type"), "application/json") {
		// parse entry as-is except hits
		err := json.NewDecoder(rq.Body).Decode(&entry)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		// use form
		err := rq.ParseForm()
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		entry.URL = rq.FormValue(formFieldService)
		entry.Template = rq.FormValue(formFieldTemplate)
	}
	err := ui.storage.Set(entry.URL, entry.Template)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}
	err = ui.engine.Reload()
	if err != nil {
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}
	wr.WriteHeader(http.StatusNoContent)
}

// correctly send JSON with required headers
func sendJSON(data interface{}, w http.ResponseWriter) {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}
