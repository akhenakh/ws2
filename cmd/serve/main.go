package main

import (
	"flag"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var (
	listen      = flag.String("listen", ":8080", "listen address")
	dir         = flag.String("dir", "htdocs", "directory to serve")
	openBrowser = flag.Bool("openBrowser", false, "open a browser while serving")
)

func main() {
	flag.Parse()
	log.Printf("listening on %s", *listen)

	cmd := ""
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "start"
	}
	if *openBrowser && cmd != "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			exec.Command(cmd, "http://127.0.0.1"+*listen).Run()
		}()
	}
	log.Fatal(http.ListenAndServe(*listen, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, ".wasm") {
			resp.Header().Set("content-type", "application/wasm")
		}

		http.FileServer(http.Dir(*dir)).ServeHTTP(resp, req)
	})))
}
