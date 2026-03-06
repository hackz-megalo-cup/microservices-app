import { AuthPanel } from '../features/auth/components/AuthPanel';
import { GatewayDemo } from '../features/gateway/components/GatewayDemo';
import { GreeterDemo } from '../features/greeter/components/GreeterDemo';

export function App() {
  return (
    <main
      style={{ maxWidth: 760, margin: '24px auto', fontFamily: 'sans-serif', padding: '0 16px' }}
    >
      <h1>connect-query Frontend Sample</h1>
      <AuthPanel />
      <GreeterDemo />
      <GatewayDemo />
    </main>
  );
}
