GOPATH := $(shell cd ../../../.. && pwd)
export GOPATH

test:
	go test -cover

save:
	godep save
