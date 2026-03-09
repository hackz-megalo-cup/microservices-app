import pRetry from 'p-retry';

const NON_RETRYABLE_CODES = [400, 401, 403, 404, 409, 422];

export async function retryWithBackoff(fn, opts = {}) {
  return pRetry(fn, {
    retries: opts.retries ?? 3,
    minTimeout: opts.minTimeout ?? 100,
    maxTimeout: opts.maxTimeout ?? 5000,
    factor: 2,
    randomize: true, // jitter
    onFailedAttempt: ({ attemptNumber, retriesLeft }) => {
      console.warn(`Attempt ${attemptNumber} failed. ${retriesLeft} retries left.`);
    },
    shouldRetry: ({ error }) => {
      const code = error?.statusCode ?? error?.response?.status ?? error?.status;
      if (code === undefined) return true;
      return !NON_RETRYABLE_CODES.includes(code);
    },
  });
}
