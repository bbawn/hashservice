package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var addr = flag.String("addr", ":8080", "http service address")
var sync = flag.Bool("sync", false, "Immediately return hashed value (step 2 of exercise)")
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

func setupShutdown() chan struct{} {
	stop := make(chan struct{})
	handler := func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("shutdownHandler")
		close(stop)
	}
	http.Handle("/shutdown", http.HandlerFunc(handler))

	return stop
}

func hashHandlerSync(w http.ResponseWriter, req *http.Request) {
	// TODO: POST only??
	// Explicitly w.Header().Set("Content-Type", ...), WriteHeader
	// Consider Request ParseForm for full validation?
	fmt.Println("delaying %v sec, password %v", delay, req.PostFormValue("password"))
	time.Sleep(time.Duration(*delay) * time.Millisecond)
	fmt.Println("delayed %v sec, password %v", delay, req.PostFormValue("password"))
	w.Write([]byte(hashAndEncode(req.PostFormValue("password"))))
}

func hashHandlerAsync(w http.ResponseWriter, req *http.Request) {
	// TODO: POST only??
	// Explicitly w.Header().Set("Content-Type", ...), WriteHeader
	if req.Method == "POST" {
		time.Sleep(time.Duration(*delay) * time.Millisecond)
		fmt.Println("delayed %v sec, password %v", delay, req.FormValue("password"))
		w.Write([]byte(hashAndEncode(req.FormValue("password"))))
	} else if req.Method == "GET" {
	}
}

func main() {
	flag.Parse()
	stop := setupShutdown()
	if *sync {
		http.Handle("/hash", http.HandlerFunc(hashHandlerSync))
	} else {
		http.Handle("/hash", http.HandlerFunc(hashHandlerAsync))
	}

	srv := startHttpServer()

	<-stop
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("INFO: hashservice: Shutdown() error: %s", err)
	}
}
