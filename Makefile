.PHONY: all serve

all: cmd/serve/serve ws2.wasm

cmd/serve/serve: cmd/serve/main.go
	go build -o cmd/serve/serve cmd/serve/main.go

ws2.wasm: main.go
	GOARCH=wasm GOOS=js go build -o htdocs/ws2.wasm ./main.go

build-server: cmd/serve/serve

all-serve: cmd/serve/serve ws2.wasm
		./cmd/serve/serve

serve: build-server
		./cmd/serve/serve

clean:
	rm -f cmd/serve/serve htdocs/ws2.wasm
