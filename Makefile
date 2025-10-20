test:
    go test ./internal/test/... -v

test-ci:
    go test ./internal/... -v -run "Test.*Unit"
