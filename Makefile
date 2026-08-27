.PHONY: test vet build fmt tidy windows desktop pack

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

build:
	./scripts/build.sh

pack: build
	@BIN=$$(ls -1 dist/tooldock-$$(go env GOOS)-$$(go env GOARCH)-* | head -1); \
	"$$BIN" pack --out dist

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 ./scripts/build.sh

desktop: build
	@BIN=$$(ls -1 dist/tooldock-$$(go env GOOS)-$$(go env GOARCH)-* | head -1); \
	"$$BIN" install-desktop; \
	"$$BIN" install-cli
