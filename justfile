version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

# Release build: static binary, no debug info, reproducible paths.
build:
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version={{version}}" -o sysutil .

run *args:
    go run . {{args}}

test:
    go vet ./...
    go test -race ./...

lint:
    pre-commit run --all-files

clean:
    rm -f sysutil
