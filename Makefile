BUILD_DIR = builds
MODULE = github.com/soerenschneider/directory-exporter
BINARY_NAME = directory-exporter
CHECKSUM_FILE = $(BUILD_DIR)/checksum.sha256
SIGNATURE_KEYFILE = ~/.signify/github.sec
DOCKER_PREFIX = ghcr.io/soerenschneider

tests:
	go test ./... -cover

clean:
	rm -rf ./$(BUILD_DIR)

build: version-info
	CGO_ENABLED=0 go build -ldflags="-X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}'" -o $(BINARY_NAME) main.go

release: clean version-info cross-build
	sha256sum $(BUILD_DIR)/directory-exporter-* > $(CHECKSUM_FILE)

signed-release: release
	pass keys/signify/github | signify -S -s $(SIGNATURE_KEYFILE) -m $(CHECKSUM_FILE)
	gh-upload-assets -o soerenschneider -r directory-exporter -f ~/.gh-token builds

cross-build: version-info
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0       go build -ldflags="-X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-x86_64    main.go
	GOOS=linux GOARCH=arm GOARM=5 CGO_ENABLED=0 go build -ldflags="-X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv5     main.go
	GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -ldflags="-X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv6     main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0       go build -ldflags="-X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-aarch64   main.go
	GOOS=openbsd GOARCH=amd64 CGO_ENABLED=0     go build -ldflags="-X 'main.BuildVersion=${VERSION}' -X 'main.CommitHash=${COMMIT_HASH}'" -o $(BUILD_DIR)/$(BINARY_NAME)-openbsd-x86_64  main.go

docker-build:
	docker build -t "$(DOCKER_PREFIX)/$(BINARY_NAME)" .

version-info:
	$(eval VERSION := $(shell git describe --tags --abbrev=0 || echo "dev"))
	$(eval COMMIT_HASH := $(shell git rev-parse HEAD))

fmt:
	find . -iname "*.go" -exec go fmt {} \;
