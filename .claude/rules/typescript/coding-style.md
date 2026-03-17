---
paths:
  - "**/*.ts"
  - "**/*.tsx"
  - "**/*.js"
  - "**/*.jsx"
---
# TypeScript/JavaScript Coding Style

> This file extends [common/coding-style.md](../common/coding-style.md) with TypeScript/JavaScript specific content.
> Based on docs/typescript-style-guide.md

## Source File Basics

- File names: **kebab-case** (`my-component.tsx`, `auth-panel.tsx`)
- Test files: `.test.ts` / `.test.tsx`
- Encoding: UTF-8, no BOM

## Declarations and Control Flow (Google TypeScript Style)

- Use `const` by default, `let` only when reassignment is required, **never** `var`
- Keep declarations scoped to the smallest possible block
- Prefer early returns to reduce nesting
- Braces are required even for single-line `if` / `for` / `while` (`useBlockStatements`)

```typescript
// BAD
if (err) return;

// GOOD
if (err) {
  return;
}
```

- Always use `===` / `!==` — never `==` / `!=` (`noDoubleEquals`)
- Use optional chaining `?.` over `&&` chains (`useOptionalChain`)

## Modules and Imports

- Use ES modules (`import` / `export`)
- **default export is prohibited** (`noDefaultExport`) — exception: config files like `vite.config.ts`
- Use `import type` for type-only imports (`useImportType`)
- Avoid `import *` (dot imports); use named imports
- Avoid TypeScript `namespace` for new code; prefer modules

Import order:
```typescript
// 1) Node / external libraries
import { useState } from "react";

// 2) Project modules
import { transport } from "@/lib/transport";
```

## Type System

- **`any` is prohibited** (`noExplicitAny`) — use `unknown` with type guards instead
- **`object` type is prohibited** — define a specific interface or type instead
- Prefer `interface` for object shapes and extension contracts
- Use `type` aliases for unions, intersections, mapped/conditional utility types
- Do not write inferrable type annotations (`noInferrableTypes`):

```typescript
// BAD
const name: string = "hello";

// GOOD
const name = "hello";
```

- Prefer type annotation `: Type` over type assertion `as Type`
- Avoid non-null assertion `!` — use explicit checks instead:

```typescript
// BAD
const el = document.getElementById("root")!;

// GOOD
const el = document.getElementById("root");
if (!el) throw new Error("root element not found");
```

- Use primitive type keywords (`string`, `number`, `boolean`) not boxed types (`String`, `Number`, `Boolean`)

## Naming

- `UpperCamelCase` for types/classes/interfaces
- `lowerCamelCase` for variables/functions/methods
- `CONSTANT_CASE` for module-level constants (`MAX_RETRY_COUNT`)
- Abbreviations: 2-char → all caps (`IO`, `DB`); 3+ char → PascalCase (`Http`, `Xml`)
- Do not prefix interface names with `I`
- Avoid underscore-prefixed private naming unless integrating with existing code conventions

## Functions

- **Top-level named functions / React components**: use `function` declaration
- **Callbacks / nested functions**: use arrow functions
- Do not use `arguments` object — use rest parameters `...args` instead
- Arrow function parameters always need parentheses (`arrowParentheses: "always"`)

```typescript
// Top-level — function declaration
export function GreeterDemo() {
  // Nested — arrow function
  const handleClick = () => { ... };
}
```

## Formatting

| Item | Setting |
|------|---------|
| Indent | 2 spaces |
| Line width | 100 characters |
| Quotes | Double quotes |
| Semicolons | Always |
| Trailing commas | Always |

## Immutability

Use spread operator for immutable updates:

```typescript
// BAD: Mutation
function updateUser(user, name) {
  user.name = name;
  return user;
}

// GOOD: Immutability
function updateUser(user, name) {
  return { ...user, name };
}
```

## Iteration and Property Access

- Prefer `for...of` for arrays and iterables
- Avoid unguarded `for...in` loops; use `Object.keys(...)` / `Object.entries(...)`
- Use optional chaining and nullish coalescing where they improve null-safety and readability

## Error Handling

Use async/await with try-catch:

```typescript
try {
  const result = await riskyOperation();
  return result;
} catch (error) {
  console.error("Operation failed:", error);
  throw new Error("Detailed user-friendly message");
}
```

## Input Validation

Use Zod for schema-based validation:

```typescript
import { z } from "zod";

const schema = z.object({
  email: z.string().email(),
  age: z.number().int().min(0).max(150),
});

const validated = schema.parse(input);
```

## Console.log

- No `console.log` statements in production code
- Use proper logging libraries instead
- See hooks for automatic detection

## Generated Code

`src/gen/` is auto-generated (protobuf) and excluded from lint/format.

## Reference

See skill: `typescript-patterns` for comprehensive TypeScript patterns.
