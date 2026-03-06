import React from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import { AppProvider } from './providers/app-provider';

async function bootstrap() {
  if (import.meta.env.VITE_USE_MOCK === 'true') {
    const { worker } = await import('../testing/browser');
    await worker.start({ onUnhandledRequest: 'bypass' });
  }

  const el = document.getElementById('root');
  if (!el) {
    throw new Error('Root element not found');
  }
  const root = createRoot(el);
  root.render(
    <React.StrictMode>
      <AppProvider>
        <App />
      </AppProvider>
    </React.StrictMode>,
  );
}

void bootstrap();
