package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
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

func hashAndEncode(s string) string {
	sBytes := []byte(s)
	hashBytes := sha512.Sum512(sBytes)
	return base64.StdEncoding.EncodeToString(hashBytes[:])
}

func startHttpServer() *http.Server {
	srv := &http.Server{Addr: *addr}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("INFO: hashservice: ListenAndServe() error: %s", err)
		}
	}()

	return srv
}

type Counter struct {
	sync.Mutex
	n int
}

func (c *Counter) next() int {
	c.Lock()
	c.n++
	c.Unlock()
	return c.n
}

var hashIdCounter Counter

type MapCache struct {
	sync.RWMutex
	m map[int]string
}

func NewMapCache() *MapCache {
	hc := new(MapCache)
	hc.m = make(map[int]string)
	return hc
}

func (hc *MapCache) Set(id int, value string) {
	hc.Lock()
	hc.m[id] = value
	hc.Unlock()
}

func (hc *MapCache) Get(id int) (string, bool) {
	hc.RLock()
	value, ok := hc.m[id]
	hc.RUnlock()
	return value, ok
}

var hashCache = NewMapCache()

func setupShutdown() chan struct{} {
	stop := make(chan struct{})
	handler := func(w http.ResponseWriter, req *http.Request) {
		log.Printf("shutdownHandler")
		close(stop)
	}
	http.Handle("/shutdown", http.HandlerFunc(handler))

	return stop
}

func doHashAsync(id int, s string) {
	time.Sleep(time.Duration(*delay) * time.Millisecond)
	hashCache.Set(id, hashAndEncode(s))
}

func hashHandlerSync(w http.ResponseWriter, req *http.Request) {
	// TODO: POST only??
	// Explicitly w.Header().Set("Content-Type", ...), WriteHeader
	// Consider Request ParseForm for full validation?
	log.Printf("delaying %v msec, password %v", *delay, req.PostFormValue("password"))
	time.Sleep(time.Duration(*delay) * time.Millisecond)
	log.Printf("delayed %v msec, password %v", *delay, req.PostFormValue("password"))
	w.Write([]byte(hashAndEncode(req.PostFormValue("password"))))
}

func hashAsyncStartHandler(w http.ResponseWriter, req *http.Request) {
	// TODO: POST only??
	// Make sure url path is exactly "/hash"?
	// Explicitly w.Header().Set("Content-Type", ...), WriteHeader
	if req.Method == "POST" {
		id := hashIdCounter.next()
		pw := req.FormValue("password")
		// XXX err
		log.Printf("async %v, password %v", id, pw)
		go doHashAsync(id, pw)
		w.Write([]byte(strconv.Itoa(id)))
	}
}

func hashAsyncFinishHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("hashAsyncFinishHandler %v", *req)
	if req.Method == "GET" {
		comps := strings.Split(req.URL.Path, "/")
		log.Printf("comps %v", comps)
		id, err := strconv.Atoi(string(comps[3]))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		hashedValue, ok := hashCache.Get(id)
		if !ok {
			msg := fmt.Sprintf("Id %d not found", id)
			http.Error(w, msg, http.StatusNotFound)
			return
		}
		w.Write([]byte(hashedValue))
	}
}

func main() {
	flag.Parse()
	stop := setupShutdown()
	http.Handle("/hash", http.HandlerFunc(hashAsyncStartHandler))
	http.Handle("/hash/id/", http.HandlerFunc(hashAsyncFinishHandler))

	// Synchronous POST endpoint from Step 2 of exercise
	http.Handle("/hashsync", http.HandlerFunc(hashHandlerSync))

	srv := startHttpServer()
	<-stop
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("INFO: hashservice: Shutdown() error: %s", err)
	}
}
