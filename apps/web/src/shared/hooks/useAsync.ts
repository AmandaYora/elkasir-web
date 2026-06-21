import { useCallback, useEffect, useState } from "react";

interface AsyncState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

// Lightweight data-fetching hook — replaces TanStack Query for read endpoints.
// Re-runs whenever `deps` change; exposes `refetch` for manual reloads and `setData`
// for optimistic/local updates.
export function useAsync<T>(fn: () => Promise<T>, deps: unknown[] = []) {
  const [state, setState] = useState<AsyncState<T>>({ data: null, loading: true, error: null });

  const refetch = useCallback(async () => {
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const data = await fn();
      setState({ data, loading: false, error: null });
    } catch (e) {
      setState({
        data: null,
        loading: false,
        error: e instanceof Error ? e.message : "Gagal memuat data.",
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => {
    void refetch();
  }, [refetch]);

  const setData = useCallback((data: T) => setState((s) => ({ ...s, data })), []);

  return { ...state, refetch, setData };
}
