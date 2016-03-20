package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
)

const (
	formFieldTemplate = "template"
	formFieldService  = "service"
	headerRedirPort   = "X-Redir-Port"
)

// description and handler of single redirection rool
type rool struct {
	// compilled template
	Pattern *template.Template
	// source template text
	content string
	// atomic counter of hits
	hits uint64
}

// description of rool for API request
type ajax struct {
	Template string `json:"template"`
	Hits     uint64 `json:"hits"`
}

// Handle http request and redirects it to URL computed by pattern and
// environment (http.Request)
func (rool *rool) Handle(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint64(&rool.hits, 1)
	url := &bytes.Buffer{}
	err := rool.Pattern.Execute(url, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Length", "0")
	if r.Method == "HEAD" {
		r.Close = true
	}
	http.Redirect(w, r, strings.TrimSpace(url.String()), http.StatusFound)
}

// container and router of rools
type roolManager struct {
	// We expect unbalanced read-write operations: reads much more then writes
	sync.RWMutex
	// collection of redirects
	pathes map[string]*rool
}

// create new manager and initialize all required internal state
func newManager() *roolManager {
	return &roolManager{pathes: make(map[string]*rool)}
}

// Handle incoming request and route to matched rool
func (manager *roolManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := getCleanPath(r)
	manager.RLock()
	defer manager.RUnlock()
	io.Copy(ioutil.Discard, r.Body)
	defer r.Body.Close()
	if rool, ok := manager.pathes[path]; ok {
		rool.Handle(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// handler of API requests
type api struct {
	Manager *roolManager
	// port of rool manager. It will be used in headers
	RedirPort string
	// JSON configuration files (as DB)
	ConfigFile string
}

// handle api request and provides CRUD operations
func (api *api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		api.get(w, r)
	case "POST":
		api.addOrUpdate(w, r)
	case "DELETE":
		api.remove(w, r)
	default:
		api.get(w, r)
	}
}

// get single (if path not set) or all services descriptions
func (api *api) get(w http.ResponseWriter, r *http.Request) {
	path := getCleanPath(r)
	api.Manager.RLock()
	defer api.Manager.RUnlock()
	w.Header().Add(headerRedirPort, api.RedirPort)
	if path == "" {
		//Get all
		ans := map[string]ajax{}
		for k, v := range api.Manager.pathes {
			ans[k] = ajax{v.content, v.hits}
		}

		sendJSON(ans, w)
	} else {
		// One
		v, ok := api.Manager.pathes[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sendJSON(v.content, w)
	}
}

// parse, buuild and add service or return error. Thread safe.
func (api *api) addService(service string, templateText string) error {
	parts := strings.Split(service, "/")
	var res []string
	for _, part := range parts {
		if len(part) != 0 {
			res = append(res, url.QueryEscape(part))
		}
	}
	service = strings.Join(res, "/")
	templ := &template.Template{}
	t, err := templ.Parse(templateText)
	if err != nil {
		return err
	}
	api.Manager.Lock()
	defer api.Manager.Unlock()
	var hits uint64
	if v, ok := api.Manager.pathes[service]; ok {
		hits = v.hits
	}
	api.Manager.pathes[service] = &rool{content: templateText, Pattern: t, hits: hits}
	return nil
}

// handle create/update request
func (api *api) addOrUpdate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = api.addService(r.FormValue(formFieldService), r.FormValue(formFieldTemplate))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	api.save()
}

// remove single rool
func (api *api) remove(w http.ResponseWriter, r *http.Request) {
	api.Manager.Lock()
	defer api.Manager.Unlock()
	delete(api.Manager.pathes, getCleanPath(r))
	api.save()
}

// get path from URL without leading and trailing slashes
func getCleanPath(r *http.Request) string {
	path := r.URL.Path
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}
	return path
}

// save current config to file and suppress errors by printing them to log
func (api *api) save() {
	r := map[string]string{}
	for k, v := range api.Manager.pathes {
		r[k] = v.content
	}
	data, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		log.Println(err)
		return
	}
	err = ioutil.WriteFile(api.ConfigFile, data, 0755)
	if err != nil {
		log.Println(err)
	}
}

// try to load config file and suppress errors by printing them to log
func (api *api) load() {
	content, err := ioutil.ReadFile(api.ConfigFile)
	if err != nil {
		log.Println(err)
		return
	}
	v := map[string]string{}
	err = json.Unmarshal(content, &v)
	if err != nil {
		log.Println(err)
		return
	}
	for k, t := range v {
		err = api.addService(k, t)
		if err != nil {
			log.Println(err)
		}
	}
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
