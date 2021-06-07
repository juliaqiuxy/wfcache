ARG GOLANG_VERSION=1.16.3

FROM golang:${GOLANG_VERSION}

WORKDIR /wfcache

RUN go get -u github.com/mitranim/gow@latest
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.40.1

COPY . .