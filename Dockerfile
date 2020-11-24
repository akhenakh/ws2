# STEP 1 build binary
FROM golang:alpine AS builder
RUN export CGO_ENABLED=0
RUN apk update && apk add --no-cache git
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY main.go /app/
COPY cmd/ /app/cmd/
COPY htdocs/ /app/htdocs/
COPY vendor/ /app/vendor/

WORKDIR /app
# Fetch dependencies.
RUN GOARCH=wasm GOOS=js go get -d -v

# Build the binary.
#RUN CGO_ENABLED=0 GOOS=linux go build -o main
RUN CGO_ENABLED=0 go build -o cmd/serve/serve cmd/serve/main.go
RUN CGO_ENABLED=0 GOARCH=wasm GOOS=js go build -o htdocs/ws2.wasm ./main.go

# STEP 2 build a small image
FROM scratch
WORKDIR /app

# Copy static executable and certificates
COPY --from=builder /app/ /app/

# Run the binary.
ENTRYPOINT ["cmd/serve/serve"]
# ENTRYPOINT ["/app/main", "--csv", "items.csv.gz"]

