GO111MODULE := on
export GO111MODULE

init:
	@go mod init

clean:
	@go mod tidy

update:
	@go get -u

test:
	@go test -v -race

build-mac:
	@cd tool && GOOS=darwin GOARCH=amd64 go build -o ../bin/mac/pcp-cli

build-linux:
	@cd tool && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../bin/linux/pcp-cli

cover:
	@go test -coverprofile=coverage.out
	@go tool cover -html=coverage.out

test-only:
	@go test -run $(CASE) -cover

.PHONY: test
