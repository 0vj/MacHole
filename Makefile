.PHONY: all build app run run-headless lint test clean

APP_NAME = MacHole

all: build

build:
	CGO_ENABLED=1 go build -ldflags="-s -w" -o $(APP_NAME) ./cmd/machole

build-cli:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(APP_NAME)-cli ./cmd/machole

app:
	./build.sh

run: build
	./$(APP_NAME)

run-headless: build
	./$(APP_NAME) -headless

lint:
	go vet ./...

test:
	go test -v ./...

clean:
	rm -rf $(APP_NAME) $(APP_NAME)-cli $(APP_NAME).app *.iconset AppIcon.icns *.dmg *.zip *.test
