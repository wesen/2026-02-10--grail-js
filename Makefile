# Lipgloss v2 beta.2 has dependency conflicts with the parent go.work
# (other modules pull in newer x/ansi). Build outside the workspace.
export GOWORK=off

.PHONY: build test bench clean

build:
	go build ./pkg/...

test:
	go test ./pkg/... -count=1 -v

bench:
	go test ./pkg/cellbuf/ -bench=. -benchmem -count=1

clean:
	go clean ./...
