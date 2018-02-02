package main

import (
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

var hashedAngryMonkey = "ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q=="

func TestPassword(t *testing.T) {
	got := hashAndEncode("angryMonkey")
	if hashedAngryMonkey != got {
		t.Errorf("expected: %v got: %v", hashedAngryMonkey, got)
	}
}

func TestSyncSimple(t *testing.T) {
	log.Printf("TestSyncSimple")
	postData := url.Values{"password": {"angryMonkey"}}
	req := httptest.NewRequest("POST", "/hash", strings.NewReader(postData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(postData.Encode())))
	w := httptest.NewRecorder()

	hashSyncHandler(w, req)

	if 200 != w.Code {
		t.Errorf("expected: %v got: %v", 200, w.Code)
	}
	if hashedAngryMonkey != w.Body.String() {
		t.Errorf("expected: %v got: %v", hashedAngryMonkey, w.Body.String())
	}
}

func TestMain(m *testing.M) {
	// Tests don't need a long delay
	*delay = 2
	os.Exit(m.Run())
}
