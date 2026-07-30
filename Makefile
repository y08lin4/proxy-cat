.PHONY: build-all build-windows build-darwin build-linux build-android build-docker test clean

FRONTEND_DIR = frontend
BINARY_NAME = proxy-cat
CMD_DIR = ./cmd/proxy-cat
DIST_DIR = dist

# Build frontend first
$(FRONTEND_DIR)/dist/index.html:
	cd $(FRONTEND_DIR) && pnpm install --frozen-lockfile && pnpm run build

# Individual platform builds
build-windows: $(FRONTEND_DIR)/dist/index.html
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

build-darwin-amd64: $(FRONTEND_DIR)/dist/index.html
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)

build-darwin-arm64: $(FRONTEND_DIR)/dist/index.html
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-darwin: build-darwin-amd64 build-darwin-arm64

build-linux-amd64: $(FRONTEND_DIR)/dist/index.html
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

build-linux-arm64: $(FRONTEND_DIR)/dist/index.html
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

build-linux: build-linux-amd64 build-linux-arm64

build-android: $(FRONTEND_DIR)/dist/index.html
	GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o $(DIST_DIR)/$(BINARY_NAME)-android-arm64 $(CMD_DIR)

build-all: build-windows build-darwin build-linux build-android

build-docker:
	docker build -t proxy-cat:latest .

test:
	go test -count=1 -timeout 60s ./...

clean:
	rm -rf $(DIST_DIR)
