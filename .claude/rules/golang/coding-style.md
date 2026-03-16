---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
# Go Coding Style

> This file extends [common/coding-style.md](../common/coding-style.md) with Go specific content.
> Based on docs/go-style-guide.md

## Style Priorities (Google Go Style)

- Prioritize **clarity**, then simplicity/conciseness
- Prefer comments that explain **why**, not restating what code already says
- Avoid unnecessary abstraction; use core language features and standard library first

## Formatting

- **gofmt** and **goimports** are mandatory — no style debates
- Keep line wrapping and spacing as produced by `gofmt`

## Naming

- Use `MixedCaps` / `mixedCaps` (no snake_case)
- Initialisms: all-caps or all-lower consistently (`URL`, `ID`, `API`, `HTTP`, `HTTPClient` — not `HttpClient`)
- Use predictable, descriptive names; avoid ambiguous abbreviations
- Package names: short, lowercase, no underscores; avoid repeating the package name in exported symbols (`greeter.Service` not `greeter.GreeterService`)
- Receiver names: 1–2 characters, consistent (typically the first letter of the type name)
- Interfaces: verb + `-er` convention (`Reader`, `Handler`)

## Imports

- Groups are ordered by `goimports`: stdlib → external → project-internal
- Blank imports (`_ "pkg"`) should be in `main` package only, not `internal`
- Dot imports (`import . "pkg"`) are prohibited

```go
import (
    // stdlib
    "context"
    "fmt"

    // external
    "connectrpc.com/connect"

    // project
    "github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)
```

## Error Handling

- Return `error` as the last return value for fallible operations
- Errors must always be checked (`errcheck` enforced)
- Ignoring errors is allowed only with explicit `_ =` and a comment explaining why
- Handle errors early and return (avoid `else` after terminal conditions)
- Use `panic` only for truly unrecoverable/impossible conditions, not normal error flow
- Error strings: start lowercase, no trailing punctuation (`revive: error-strings`)
- Custom error types must end with `...Error` (`errname` enforced)
- Compare errors with `errors.Is()` / `errors.As()` — never use `==` directly (`errorlint` enforced)
- Wrap errors with `%w` (not `%v`) to preserve the error chain:

```go
// BAD
return fmt.Errorf("db query failed: %v", err)

// GOOD
return fmt.Errorf("db query failed: %w", err)
```

## Structs and Initialization

- Always use field-named struct literals:

```go
// BAD
srv := &http.Server{"", handler, 30 * time.Second}

// GOOD
srv := &http.Server{
    Handler:     handler,
    ReadTimeout: 30 * time.Second,
}
```

- Prefer `nil` over empty slice literals (`return nil` not `return []string{}`)

## Concurrency

- goroutine lifetime must always be clear — use `context` for cancellation
- Prefer `sync.WaitGroup` or `errgroup` over fire-and-forget goroutines
- Pass `context.Context` as the first argument to every function; use `context.Background()` only in `main` or top-level entry points

## Function Design

- Keep functions to ~60 lines (roughly one screen)
- Extract helpers when functions grow beyond that
- Do not use naked returns; always return values explicitly
- Named return values are for documentation purposes only

```go
// BAD — naked return
func find(id int) (result string, err error) {
    return
}

// GOOD
func find(id int) (string, error) {
    return result, nil
}
```

## Design Principles

- Keep APIs simple and explicit
- Accept interfaces, return structs
- Keep interfaces small (1-3 methods)

## Generated Code

`gen/` is auto-generated (protobuf/connect-go) and excluded from lint.

## Reference

See skill: `golang-patterns` for comprehensive Go idioms and patterns.
