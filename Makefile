
deps:
	# as of 0.4.1 we can't install aws sdk via dep , just keep it simple
	go get github.com/aws/aws-sdk-go
	go get github.com/aws/aws-sdk-go/service/codecommit
	go get github.com/aws/aws-sdk-go/aws/session
	go get github.com/urfave/cli


bin/gitmirror: deps
	go build -o bin/gitmirror .

run: bin/gitmirror
	cat repos.txt | ./bin/gitmirror

clean:
	rm -rf bin
