---
paths:
  - "**/*.ts"
  - "**/*.tsx"
  - "**/*.js"
  - "**/*.jsx"
---
# TypeScript/JavaScript Patterns

> This file extends [common/patterns.md](../common/patterns.md) with TypeScript/JavaScript specific content.
> Frontend architecture based on docs/bulletproof-react.md

## Frontend Directory Structure (bulletproof-react)

```
src/
├── app/                    # Application entry point
│   ├── main.tsx            # Bootstrap (MSW init, ReactDOM.render)
│   ├── App.tsx             # Root component
│   └── providers/
│       └── app-provider.tsx # TransportProvider + QueryClientProvider
├── features/               # Feature modules
│   └── <feature>/
│       ├── api/            # API call logic
│       ├── components/     # UI components
│       ├── hooks/          # Custom hooks (state + API wrappers)
│       └── types/          # Type definitions
├── gen/                    # Auto-generated code (buf generate)
├── interceptors/           # connect-go interceptors (auth token injection)
├── lib/                    # Shared utilities
│   ├── transport.ts        # Connect transport config
│   └── query-client.ts     # React Query client config
├── testing/                # Test utilities
│   ├── browser.ts          # MSW worker
│   ├── handlers.ts         # MSW handlers
│   └── test-utils.tsx      # Test provider wrappers
└── types/                  # Shared type definitions
```

## Feature-First Design Principles

- Each feature is an independent module
- **Direct imports between features are prohibited** — shared code goes in `lib/` or `types/`
- Each feature has `api/`, `components/`, `hooks/`, `types/` subdirectories

### api/

- External API communication logic
- REST APIs: use `fetch`
- gRPC APIs: connect-query handles the API layer

### components/

- UI components
- Depend only on `hooks/` and `api/` — never touch `fetch` or `transport` directly

### hooks/

- Custom hooks
- auth: `useAuth()` wraps state management + API calls
- gRPC features: connect-query `useQuery` / `useMutation` serve as the hooks pattern

### types/

- TypeScript type definitions
- API response types, component prop types

## connect-query Integration

### Query Pattern (read)

```typescript
useQuery(greet, { name }, { enabled: false }) // manual trigger
```

### Mutation Pattern (write)

```typescript
useMutation + createClient // with idempotency-key
```

## Adding a New Feature

1. Create `src/features/<feature-name>/` directory
2. Define types in `types/index.ts`
3. Add API logic in `api/` (REST → fetch, gRPC → connect-query)
4. Add custom hooks in `hooks/`
5. Add UI components in `components/`
6. Wire component into `App.tsx`

## API Response Format

```typescript
interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  meta?: {
    total: number;
    page: number;
    limit: number;
  };
}
```

## Custom Hooks Pattern

```typescript
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => setDebouncedValue(value), delay);
    return () => clearTimeout(handler);
  }, [value, delay]);

  return debouncedValue;
}
```

## Repository Pattern

```typescript
interface Repository<T> {
  findAll(filters?: Filters): Promise<T[]>;
  findById(id: string): Promise<T | null>;
  create(data: CreateDto): Promise<T>;
  update(id: string, data: UpdateDto): Promise<T>;
  delete(id: string): Promise<void>;
}
```
