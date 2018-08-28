package main

import (
	"flag"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

var (
	listen = flag.String("listen", ":8080", "listen address")
	dir    = flag.String("dir", "htdocs", "directory to serve")
)

func main() {
	flag.Parse()
	log.Printf("listening on %s", *listen)

	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("open", "http://127.0.0.1"+*listen).Run()
	}()
	log.Fatal(http.ListenAndServe(*listen, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, ".wasm") {
			resp.Header().Set("content-type", "application/wasm")
		}

		http.FileServer(http.Dir(*dir)).ServeHTTP(resp, req)
	})))
}
