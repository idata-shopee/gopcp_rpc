GOPATH := $(shell cd ../../../.. && pwd)
export GOPATH

test:
	go test -cover

test-only:
	go test -run $(CASE) -cover


save:
	godep save
