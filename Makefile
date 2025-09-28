PKG=./...
CLI_BIN=tdb
CLI_SRC=./cmd/tdb
GIT_TAG:=$(shell git describe --tags --abbrev=0 2>/dev/null)
GIT_COMMIT:=$(shell git rev-parse --short HEAD 2>/dev/null)
CLI_VERSION?=$(if $(GIT_COMMIT),$(if $(GIT_TAG),$(GIT_TAG)-$(GIT_COMMIT),$(GIT_COMMIT)),dev)
CLI_VERSION_SAFE:=$(subst /,-,$(CLI_VERSION))
CLI_OUT_DIR=dist
CLI_PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: release

run:
	go run cmd/tdb/main.go

test:
	go test -count=1 $(PKG)

format:
	go fmt $(PKG)

vet:
	go vet $(PKG)

.PHONY: release
release: clean
	@mkdir -p $(CLI_OUT_DIR)
	@for platform in $(CLI_PLATFORMS); do \
		GOOS=$${platform%%/*}; \
		GOARCH=$${platform##*/}; \
		EXT=""; \
		if [ $$GOOS = "windows" ]; then EXT=.exe; fi; \
		OUT_DIR=$(CLI_OUT_DIR)/$${GOOS}_$${GOARCH}; \
		mkdir -p $$OUT_DIR; \
		BIN_NAME=$(CLI_BIN)$$EXT; \
		ARCHIVE_NAME=tdb-cli_$(CLI_VERSION_SAFE)_$${GOOS}_$${GOARCH}; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH \
			go build -trimpath -ldflags "-s -w -X github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/version.Version=$(CLI_VERSION)" \
			-o $$OUT_DIR/$$BIN_NAME $(CLI_SRC); \
		case $$GOOS in \
			windows|darwin) \
				(cd $$OUT_DIR && zip -q ../$$ARCHIVE_NAME.zip $$BIN_NAME); \
				;; \
			*) \
				(cd $$OUT_DIR && tar -czf ../$$ARCHIVE_NAME.tar.gz $$BIN_NAME); \
				;; \
		esac; \
	done

.PHONY: clean
clean:
	rm -rf $(CLI_OUT_DIR)