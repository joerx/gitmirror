# build stage
FROM golang:alpine AS builder
ENV GOPATH=/go OUTPUT=/gitmirror
RUN apk --no-cache add make git
# cache go deps before building
ADD Makefile /go/src/github.com/joerx/gitmirror/Makefile
WORKDIR /go/src/github.com/joerx/gitmirror
RUN make deps
# build binary
ADD . /go/src/github.com/joerx/gitmirror
RUN make

# final stage
FROM alpine
COPY --from=builder /gitmirror /
ENTRYPOINT ["/gitmirror"]
