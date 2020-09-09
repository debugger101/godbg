build:
	@go build -o godbg ./

install:
	@go install ./

uninstall:
	@go clean -i github.com/debugger101/godbg

test:
	@go test -v
