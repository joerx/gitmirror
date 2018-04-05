
default: build

deps:
	# as of 0.4.1 we can't install aws sdk via dep , just keep it simple
	go get github.com/aws/aws-sdk-go
	go get github.com/aws/aws-sdk-go/service/codecommit
	go get github.com/aws/aws-sdk-go/aws/session
	go get github.com/urfave/cli
	go get github.com/google/go-github/github
	go get golang.org/x/oauth2


build: deps
	go build -o bin/gitmirror .

run: build
	./bin/gitmirror

clean:
	rm -rf bin
