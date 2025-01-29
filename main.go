package main

import (
	"fmt"
	"log"
	"net/http"
	rushcache "rushcache"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	rushcache.NewGroup("scores", 2<<10, rushcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not found", key)
		}))

	addr := "localhost:9999"
	peers := rushcache.NewHTTPPool(addr)
	log.Println("rushcache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
