---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go Testing

> This file extends [common/testing.md](../common/testing.md) with Go specific content.
> Based on docs/go-style-guide.md

## Framework

Use the standard `go test` with **table-driven tests**.

## Table-Driven Tests

```go
tests := []struct {
    name  string
    input string
    want  string
}{
    {name: "empty", input: "", want: "Hello, World!"},
    {name: "with name", input: "Go", want: "Hello, Go!"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := greet(tt.input)
        if got != tt.want {
            t.Errorf("greet(%q) = %q, want %q", tt.input, got, tt.want)
        }
    })
}
```

## Test Helpers

Test helper functions must call `t.Helper()` at the top:

```go
func setupTestDB(t *testing.T) *pgxpool.Pool {
    t.Helper()
    // ...
}
```

## Assertion and Failure Style

- Failure messages should include function/action, input, and expected vs actual values
- Prefer `t.Error` / `t.Errorf` to keep running the rest of the table when possible
- Use `t.Fatal` / `t.Fatalf` only when the current test cannot continue safely
- For errors, use `errors.Is` / `errors.As` over brittle string matching

## Race Detection

Always run with the `-race` flag:

```bash
go test -race ./...
```

## Coverage

```bash
go test -cover ./...
```

## Reference

See skill: `golang-testing` for detailed Go testing patterns and helpers.
