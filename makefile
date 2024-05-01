.PHONY: all build_linux_amd64 build_linux_arm

BINARY_NAME=v6-proxy

all: build_linux_amd64 build_linux_arm

build_linux_amd64:
	GOOS=linux GOARCH=amd64 go build -o ${BINARY_NAME}-linux-amd64 main.go

build_linux_arm:
	GOOS=linux GOARCH=arm64 go build -o ${BINARY_NAME}-linux-arm64 main.go

clean:
	rm -f ${BINARY_NAME}-linux-amd64
	rm -f ${BINARY_NAME}-linux-arm64
