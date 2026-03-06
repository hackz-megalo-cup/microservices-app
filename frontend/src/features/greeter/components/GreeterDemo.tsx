import { useQuery } from '@connectrpc/connect-query';
import { useState } from 'react';
import { greet } from '../../../gen/greeter/v1/greeter-GreeterService_connectquery';

export function GreeterDemo() {
  const [name, setName] = useState('World');

  const greetQuery = useQuery(
    greet,
    { name },
    {
      enabled: false,
    },
  );

  return (
    <section style={{ border: '1px solid #ddd', padding: 16, borderRadius: 8 }}>
      <h2>
        Greeter RPC Demo <span style={{ fontSize: 12, color: '#666' }}>→ greeter_db.greetings</span>
      </h2>
      <p>RPC 呼び出し時に greeter が同期的に greetings テーブルへ INSERT します。</p>

      <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="name"
          style={{ padding: 8 }}
        />
        <button
          type="button"
          onClick={() => void greetQuery.refetch()}
          disabled={greetQuery.isFetching}
        >
          {greetQuery.isFetching ? 'Sending...' : 'Send Request'}
        </button>
      </div>

      {greetQuery.data && (
        <pre style={{ marginTop: 12, background: '#f7f7f7', padding: 12 }}>
          {JSON.stringify(greetQuery.data, null, 2)}
        </pre>
      )}

      {greetQuery.error && (
        <pre style={{ marginTop: 12, color: '#b00020' }}>{greetQuery.error.message}</pre>
      )}
    </section>
  );
}
