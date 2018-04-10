IMAGE_ORG ?= $(shell git config --get remote.origin.url | awk -F":|/" '{print $$2}' | cut -d"." -f1)
IMAGE_NAME := $(IMAGE_ORG)/gitmirror
OUTPUT ?= ./bin/gitmirror

default: clean build

deps:
	go get github.com/aws/aws-sdk-go
	go get github.com/aws/aws-sdk-go/service/codecommit
	go get github.com/aws/aws-sdk-go/aws/session
	go get github.com/urfave/cli
	go get github.com/google/go-github/github
	go get golang.org/x/oauth2


build: deps
	go build -o $(OUTPUT) .

run: build
	$(OUTPUT)

clean:
	rm -rf bin

image:
	docker build -t $(IMAGE_NAME):latest .
