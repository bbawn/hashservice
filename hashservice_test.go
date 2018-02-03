package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests don't need a long delay
const TestDelayMsec = 2

var hashedAngryMonkey = "ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q=="

func handlerGet(f http.HandlerFunc, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	f(w, req)

	return w
}

func handlerPost(f http.HandlerFunc, path string, postData url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(postData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(postData.Encode())))
	w := httptest.NewRecorder()
	f(w, req)

	return w
}

func TestHashAndEncode(t *testing.T) {
	got := hashAndEncode("angryMonkey")
	if hashedAngryMonkey != got {
		t.Errorf("expected: %v got: %v", hashedAngryMonkey, got)
	}
}

func TestSyncSimple(t *testing.T) {
	samples := []string{"angryMonkey", "", "1", "angryMonkey1"}

	for _, pw := range samples {
		postData := url.Values{"password": {pw}}
		w := handlerPost(http.HandlerFunc(hashSyncHandler), "/hashsync", postData)
		if http.StatusOK != w.Code {
			t.Errorf("expected: %v got: %v", http.StatusOK, w.Code)
		}
		expected := hashAndEncode(pw)
		if expected != w.Body.String() {
			t.Errorf("expected: %v got: %v", expected, w.Body.String())
		}
	}
}

func TestAsyncSimple(t *testing.T) {
	reset() // Need fresh stats

	// Post password
	postData := url.Values{"password": {"angryMonkey"}}
	w := handlerPost(http.HandlerFunc(hashAsyncStartHandler), "/hash", postData)
	if http.StatusOK != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusOK, w.Code)
	}

	id, err := strconv.Atoi(w.Body.String())
	if err != nil {
		t.Errorf("expected: integer got: %v", w.Body.String())
	}

	// Sleep long enough for the hashing to complete
	// NOTE: non-deterministic, GET could fail on a heavily-loaded system
	time.Sleep(10 * time.Duration(TestDelayMsec) * time.Millisecond)

	// Retrieve password by id
	path := fmt.Sprintf("/hash/id/%d", id)
	w = handlerGet(http.HandlerFunc(hashAsyncFinishHandler), path)
	if http.StatusOK != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusOK, w.Code)
	}
	if hashedAngryMonkey != w.Body.String() {
		t.Errorf("expected: %v got: %v", hashedAngryMonkey, w.Body.String())
	}

	// Get stats
	w = handlerGet(http.HandlerFunc(statsHandler), path)
	if http.StatusOK != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusOK, w.Code)
	}

	var s stats
	err = json.Unmarshal([]byte(w.Body.String()), &s)
	if err != nil {
		t.Errorf("expected: json got: %v", w.Body.String())
	}
	if s.Total != 1 {
		t.Errorf("expected: %v got: %v", 1, s.Total)
	}

	// Shutdown
	// XXX don't know how to easily test that remaining requests complete
	h, stop := setupShutdown()
	w = handlerGet(h, path)
	select {
	case <-stop:
	default:
		t.Errorf("expected: stop signal")
	}
}

func TestInvalidReqs(t *testing.T) {

	// Invalid Methods
	w := handlerPost(http.HandlerFunc(hashAsyncFinishHandler), "/hash/id/1", nil)
	if http.StatusMethodNotAllowed != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusMethodNotAllowed, w.Code)
	}

	w = handlerPost(http.HandlerFunc(statsHandler), "/stats", nil)
	if http.StatusMethodNotAllowed != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusMethodNotAllowed, w.Code)
	}

	w = handlerPost(http.HandlerFunc(hashAsyncFinishHandler), "/shutdown", nil)
	if http.StatusMethodNotAllowed != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusMethodNotAllowed, w.Code)
	}

	w = handlerGet(http.HandlerFunc(hashSyncHandler), "/hash")
	if http.StatusMethodNotAllowed != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusMethodNotAllowed, w.Code)
	}

	// Invalid paths
	w = handlerGet(http.HandlerFunc(hashAsyncFinishHandler), "/hash/id/1/foo")
	if http.StatusBadRequest != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusBadRequest, w.Code)
	}

	w = handlerGet(http.HandlerFunc(hashAsyncFinishHandler), "/hash/id/foo")
	if http.StatusBadRequest != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusBadRequest, w.Code)
	}

	w = handlerGet(http.HandlerFunc(hashAsyncFinishHandler), "/hash/id/12345678")
	if http.StatusNotFound != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusNotFound, w.Code)
	}

	// Invalid parameters
	postData := url.Values{"foo": {"angryMonkey"}}
	w = handlerPost(http.HandlerFunc(hashAsyncStartHandler), "/hash", postData)
	if http.StatusBadRequest != w.Code {
		t.Errorf("expected: %v got: %v", http.StatusOK, w.Code)
	}
}

// Stress and benchmark server by running a lot of concurrent requests
func BenchmarkSimple(b *testing.B) {
	mux, _ := newServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := ts.Client()
	notFoundCount := 0
	var wg sync.WaitGroup
	var i int
	reset()

	hashRequestor := func() {
		defer wg.Done()
		pw := fmt.Sprintf("angryMonkey-%d-of-%d", i+1, b.N)
		postData := url.Values{"password": {pw}}
		resp, err := client.Post(ts.URL+"/hash",
			"application/x-www-form-urlencoded",
			strings.NewReader(postData.Encode()))
		if err != nil {
			b.Errorf("POST err: %v", err)
		}

		bod, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			b.Errorf("ReadAll err: %v", err)
		}
		resp.Body.Close()
		id, err := strconv.Atoi(string(bod))
		if err != nil || id > b.N {
			b.Errorf("Invalid id: %d err: %v", id, err)
		}
	}

	hashRetriever := func() {
		defer wg.Done()

		id := rand.Intn(i + 1)
		u := fmt.Sprintf("%s/hash/id/%d", ts.URL, id)
		resp, err := client.Get(u)
		if err != nil {
			b.Errorf("GET err: %v", err)
		}

		// Some ids will not be available yet: not found is OK
		if resp.StatusCode == http.StatusNotFound {
			notFoundCount++
		} else if resp.StatusCode != http.StatusOK {
			b.Errorf("GET statusCode: %d", resp.StatusCode)
		}

		resp.Body.Close()
	}

	for i = 0; i < b.N; i++ {
		wg.Add(1)
		go hashRequestor()
		wg.Add(1)
		go hashRetriever()

		// For large enough N (~200), we run out of socket descriptors
		// if we don't wait here. I think concurrent Client Do() calls
		// can't share TCP connections (makes sense, see http Client
		// Do() doc). Work around with a rendezvous point.
		if i%100 == 0 {
			wg.Wait()
		}
	}

	// Get stats before all requestors finish
	resp, err := client.Get(ts.URL + "/stats")
	if err != nil || resp.StatusCode != http.StatusOK {
		b.Errorf("stats GET err: %v statusCode %d", err, resp.StatusCode)
	}
	var s stats
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&s)
	if err != nil {
		b.Errorf("expected: json Decode: %v", err)
	}
	wg.Wait()
	log.Printf("BenchmarkSimple b.N %d notFoundCount %d stats total %d average %v ms",
		b.N, notFoundCount, s.Total, s.Average)
}

func TestMain(m *testing.M) {
	*delay = TestDelayMsec
	os.Exit(m.Run())
}
