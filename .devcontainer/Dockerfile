FROM golang:latest

WORKDIR /workspace

RUN go install -v golang.org/x/tools/gopls@latest
RUN go mod download