.PHONY: test vet build fmt tidy windows desktop

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

build:
	go build -o devtoolbox .

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/devtoolbox-windows-amd64.exe .

tidy:
	go mod tidy

desktop: build
	./devtoolbox install-desktop
	./devtoolbox install-cli
