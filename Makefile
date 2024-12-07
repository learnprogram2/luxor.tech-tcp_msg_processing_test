
build-server:
	@echo "Building server binary..."
	go build  -o output/server ./cmd/server/main.go
	@echo "success"

build-server-linux:
	@echo "Building server binary..."
	GOOS=linux GOARCH=amd64 go build -o output/server_linux ./cmd/server/main.go
	@echo "success"

build-client:
	@echo "Building client binary..."
	go build  -o output/client ./cmd/client/main.go
	@echo "success"
build-client-linux:
	@echo "Building client binary..."
	GOOS=linux GOARCH=amd64 go build -o output/client_linux ./cmd/client/main.go
	@echo "success"


run-server:
	@echo "Running server binary..."
	go run ./cmd/server/main.go
	@echo "success"
run-client:
	@echo "Running client binary..."
	go run ./cmd/client/main.go
	@echo "success"