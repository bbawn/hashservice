package main

import (
	"crypto/sha512"
	"encoding/base64"
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func hashAndEncode(s string) string {
	sBytes := []byte(s)
	hashBytes := sha512.Sum512(sBytes)
	return base64.StdEncoding.EncodeToString(hashBytes[:])
}

func main() {
	flag.Parse()
	http.Handle("/hash", http.HandlerFunc(hashHandler))
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("FATAL: hashservice: ListenAndServe: ", err)
	}
}

func hashHandler(w http.ResponseWriter, req *http.Request) {
	// TODO: POST only??
	// Explicitly w.Header().Set("Content-Type", ...), WriteHeader
	w.Write([]byte(hashAndEncode(req.FormValue("password"))))
}
