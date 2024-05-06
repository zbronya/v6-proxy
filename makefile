.PHONY: all build_linux_amd64 build_linux_arm

BINARY_NAME=v6-proxy
CGO_ENABLED=0

all: build_linux_amd64 build_linux_arm

build_linux_amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -o ${BINARY_NAME}-linux-amd64 main.go

build_linux_arm:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build -o ${BINARY_NAME}-linux-arm64 main.go

clean:
	rm -f ${BINARY_NAME}-linux-amd64
	rm -f ${BINARY_NAME}-linux-arm64
