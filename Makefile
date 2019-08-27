build:
	@go build -o godbg ./

install:
	@go install ./

uninstall:
	@go clean -i github.com/chainhelen/godbg

test:
	@go test -v
