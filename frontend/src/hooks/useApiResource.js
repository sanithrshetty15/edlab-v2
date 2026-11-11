import { useCallback, useEffect, useEffectEvent, useRef, useState } from "react";

export function useApiResource(fetcher, key = "default", options = {}) {
  const { enabled = true, initialData = null } = options;
  const [version, setVersion] = useState(0);
  const [state, setState] = useState(() => ({
    data: initialData,
    error: null,
    loading: Boolean(enabled),
    requestKey: `${key}:${enabled ? "on" : "off"}:0`,
  }));
  const requestKey = `${key}:${enabled ? "on" : "off"}:${version}`;
  const stale = state.requestKey !== requestKey;
  const load = useEffectEvent(() => fetcher());

  useEffect(() => {
    if (!enabled) return undefined;

    let cancelled = false;
    Promise.resolve()
      .then(load)
      .then((result) => {
        if (cancelled) return;
        setState({
          data: result,
          error: null,
          loading: false,
          requestKey,
        });
      })
      .catch((requestError) => {
        if (cancelled) return;
        setState((current) => ({
          ...current,
          error: requestError,
          loading: false,
          requestKey,
        }));
      });

    return () => {
      cancelled = true;
    };
  }, [enabled, requestKey]);

  const reload = useCallback(() => setVersion((current) => current + 1), []);

  const setData = useCallback((updater) => {
    setState((current) => ({
      ...current,
      data:
        typeof updater === "function" ? updater(current.data) : updater,
    }));
  }, []);

  return {
    data: state.data,
    error: stale ? null : state.error,
    loading: enabled && (state.loading || stale),
    reload,
    setData,
  };
}

export function useMutation(mutation) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const reset = useCallback(() => {
    setError(null);
    setSuccess(null);
  }, []);

  const run = useCallback(
    async (...args) => {
      setLoading(true);
      setError(null);
      setSuccess(null);
      try {
        const result = await mutation(...args);
        if (mountedRef.current) setSuccess(result);
        return result;
      } catch (requestError) {
        if (mountedRef.current) setError(requestError);
        throw requestError;
      } finally {
        if (mountedRef.current) setLoading(false);
      }
    },
    [mutation],
  );

  return { run, loading, error, success, reset, setError };
}
