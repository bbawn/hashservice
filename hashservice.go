package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var addr = flag.String("addr", ":8080", "http service address")
var delay = flag.Int("delay", 5000, "Milliseconds to delay hashing")

type stats struct {
	Total   int     `json:"total"`
	Average float64 `json:"average"`
}

func hashAndEncode(s string) string {
	sBytes := []byte(s)
	hashBytes := sha512.Sum512(sBytes)
	return base64.StdEncoding.EncodeToString(hashBytes[:])
}

// Generate sequential IDs
type counter struct {
	sync.Mutex
	n int
}

func (c *counter) next() int {
	c.Lock()
	c.n++
	result := c.n
	c.Unlock()
	return result
}

func (c *counter) reset() {
	c.Lock()
	c.n = 0
	c.Unlock()
}

var hashIdCounter counter

// Store hashed/encoded values, addressable by ID
type mapCache struct {
	sync.RWMutex
	m map[int]string

	// Sum of all processing times for element of m
	totalTime time.Duration
}

func newMapCache() *mapCache {
	mc := new(mapCache)
	mc.m = make(map[int]string)
	return mc
}

func (mc *mapCache) set(id int, value string, startTime time.Time) {
	mc.Lock()
	mc.m[id] = value
	procTime := time.Now().Sub(startTime)
	mc.totalTime += procTime
	mc.Unlock()
}

func (mc *mapCache) get(id int) (string, bool) {
	mc.RLock()
	value, ok := mc.m[id]
	mc.RUnlock()
	return value, ok
}

func (mc *mapCache) getStats() (int64, time.Duration) {
	mc.RLock()
	count := len(mc.m)
	totalTime := mc.totalTime
	mc.RUnlock()
	return int64(count), totalTime
}

func (mc *mapCache) reset() {
	mc.Lock()
	mc.m = make(map[int]string)
	mc.totalTime = time.Duration(0)
	mc.Unlock()
}

var hashCache = newMapCache()

// Re-initialize state for unit tests
func reset() {
	hashCache.reset()
	hashIdCounter.reset()
}

func doHashAsync(id int, s string) {
	time.Sleep(time.Duration(*delay) * time.Millisecond)
	startTime := time.Now()
	hashCache.set(id, hashAndEncode(s), startTime)
}

func setupShutdown() (http.HandlerFunc, chan struct{}) {
	stop := make(chan struct{})
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			http.Error(w, "GET method is required", http.StatusMethodNotAllowed)
			return
		}
		close(stop)
	}

	return handler, stop
}

// Hash/encode password and return it in response
func hashSyncHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, "POST method is required", http.StatusMethodNotAllowed)
		return
	}

	time.Sleep(time.Duration(*delay) * time.Millisecond)
	req.ParseForm()
	pw, ok := req.PostForm["password"]
	if !ok {
		http.Error(w, "password parameter is required", http.StatusBadRequest)
		return
	}
	w.Write([]byte(hashAndEncode(pw[0])))
}

// Schedule hashing of password, return ID to be used for retrieval
func hashAsyncStartHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, "POST method is required", http.StatusMethodNotAllowed)
		return
	}

	req.ParseForm()
	pw, ok := req.PostForm["password"]
	if !ok {
		http.Error(w, "password parameter is required", http.StatusBadRequest)
		return
	}

	id := hashIdCounter.next()
	go doHashAsync(id, pw[0])
	w.Write([]byte(strconv.Itoa(id)))
}

// Retrieve a hashed password using its ID
func hashAsyncFinishHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "GET method is required", http.StatusMethodNotAllowed)
		return
	}

	comps := strings.Split(req.URL.Path, "/")
	if len(comps) != 3 {
		http.Error(w, "Invalid path: "+req.URL.Path, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(string(comps[2]))
	if err != nil {
		http.Error(w, "Invalid ID: path: "+req.URL.Path, http.StatusBadRequest)
		return
	}

	hashedValue, ok := hashCache.get(id)
	if !ok {
		msg := fmt.Sprintf("Id %d not found", id)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	w.Write([]byte(hashedValue))
}

func statsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "GET method is required", http.StatusMethodNotAllowed)
		return
	}

	count, totalTime := hashCache.getStats()
	var avg float64
	if count != 0 {
		avg = float64(totalTime) / (float64(count * int64(time.Millisecond)))
	}
	statsJson, err := json.Marshal(stats{int(count), avg})
	if err != nil {
		msg := fmt.Sprintf("json error: %v count: %v avg: %v", err, count, avg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.Write(statsJson)
}

func newServeMux() (*http.ServeMux, chan struct{}) {
	mux := http.NewServeMux()
	shutdownHandler, stop := setupShutdown()
	mux.Handle("/shutdown", shutdownHandler)
	mux.Handle("/hash", http.HandlerFunc(hashAsyncStartHandler))
	mux.Handle("/hash/", http.HandlerFunc(hashAsyncFinishHandler))
	mux.Handle("/stats", http.HandlerFunc(statsHandler))

	// Synchronous POST endpoint from Step 2 of exercise
	mux.Handle("/hashsync", http.HandlerFunc(hashSyncHandler))

	return mux, stop
}

func main() {
	flag.Parse()
	mux, stop := newServeMux()
	srv := &http.Server{Addr: *addr, Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("INFO: hashservice: ListenAndServe(): %s", err)
		}
	}()

	<-stop
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("INFO: hashservice: Shutdown() error: %s", err)
	}
}
