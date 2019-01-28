package redirect

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// Simple single-file storage. All rules saved as-is by JSON indented encoder to the provided file after each Set ops.
type JsonStorage struct {
	FileName string // File name to store and read
	cache    map[string]string
	lock     sync.RWMutex
}

// Set or replace one rule, serialize cache to JSON and then dump to disk. Even if dump failed rule is saved into cache
func (js *JsonStorage) Set(url string, locationTemplate string) error {
	js.lock.Lock()
	defer js.lock.Unlock()
	if js.cache == nil {
		js.cache = make(map[string]string)
	}
	js.cache[url] = locationTemplate
	return js.unsafeDump()
}

// Get single record from cache
func (js *JsonStorage) Get(url string) (string, bool) {
	js.lock.RLock()
	defer js.lock.RUnlock()
	v, ok := js.cache[url]
	return v, ok
}

// Remove rule from cache and save dump to disk. Even if dump failed rule removed from cache
func (js *JsonStorage) Remove(url string) error {
	if js.cache == nil {
		return nil
	}
	js.lock.Lock()
	defer js.lock.Unlock()
	delete(js.cache, url)
	return js.unsafeDump()
}

// All rules stored in cache. Never returns error
func (js *JsonStorage) All() ([]*Rule, error) {
	var ans = make([]*Rule, 0, len(js.cache))
	js.lock.RLock()
	defer js.lock.RUnlock()
	for url, location := range js.cache {
		ans = append(ans, &Rule{
			URL:              url,
			LocationTemplate: location,
		})
	}
	return ans, nil
}

// Read all rules from file. Will not update cache if file will not exists
func (js *JsonStorage) Reload() error {
	js.lock.RLock() // prevent read and write the same file
	data, err := ioutil.ReadFile(js.FileName)
	js.lock.RUnlock()
	if os.IsNotExist(err) {
		// nothing to reload
		log.Println(js.FileName, err)
		return nil
	} else if err != nil {
		return err
	}
	var cache map[string]string
	err = json.Unmarshal(data, &cache)
	if err != nil {
		// failed to decode json - mb broken?
		return err
	}
	js.lock.Lock()
	js.cache = cache
	js.lock.Unlock()
	return nil
}

func (js *JsonStorage) unsafeDump() error {
	data, err := json.MarshalIndent(js.cache, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(js.FileName, data, 0755)
}
