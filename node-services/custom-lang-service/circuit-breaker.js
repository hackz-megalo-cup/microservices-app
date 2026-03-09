import CircuitBreaker from 'opossum';

const DEFAULT_OPTIONS = {
  timeout: 3000,
  errorThresholdPercentage: 50,
  resetTimeout: 30000,
  volumeThreshold: 5,
};

export function createCircuitBreaker(fn, opts = {}) {
  const breaker = new CircuitBreaker(fn, { ...DEFAULT_OPTIONS, ...opts });
  breaker.on('open', () => console.warn(`Circuit breaker OPEN: ${breaker.name}`));
  breaker.on('halfOpen', () => console.info(`Circuit breaker HALF-OPEN: ${breaker.name}`));
  breaker.on('close', () => console.info(`Circuit breaker CLOSED: ${breaker.name}`));
  breaker.fallback(() => {
    throw new Error('Service unavailable (circuit breaker open)');
  });
  return breaker;
}
