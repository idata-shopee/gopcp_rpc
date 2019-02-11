GOPATH := $(shell cd ../../../.. && pwd)
export GOPATH

test:
	go test -cover

restore:
	godep restore -v

test-only:
	go test -run $(CASE) -cover

save:
	godep save
