export async function fetchWithTimeout(input: URL | string, init: RequestInit = {}, timeoutMs = 5000): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  let signal: AbortSignal = controller.signal;
  if (init.signal) {
    if (typeof AbortSignal !== 'undefined' && typeof AbortSignal.any === 'function') {
      signal = AbortSignal.any([init.signal, controller.signal]);
    } else {
      if (init.signal.aborted) controller.abort();
      init.signal.addEventListener('abort', () => controller.abort(), { once: true });
    }
  }

  try {
    return await fetch(input, { ...init, signal });
  } finally {
    clearTimeout(timer);
  }
}
