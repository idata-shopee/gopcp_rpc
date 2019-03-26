GOPATH := $(shell cd ../../../.. && pwd)
export GOPATH

init-dep:
	@dep init

status-dep:
	@dep status

ensure-dep:
	@dep ensure

test:
	@go test -v -race

build-mac:
	@cd tool && GOOS=darwin GOARCH=amd64 go build -o ../bin/mac/pcp-cli

cover:
	@go test -coverprofile=coverage.out
	@go tool cover -html=coverage.out

restore:
	@godep restore -v

test-only:
	@go test -run $(CASE) -cover

save:
	@godep save
