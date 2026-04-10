"use client";

import { useEffect, useState, type ComponentType, type ReactNode } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { createQueryClient } from "./query-client";

export function QueryProvider({ children, showDevtools = true }: { children: ReactNode; showDevtools?: boolean }) {
  const [queryClient] = useState(createQueryClient);
  const [Devtools, setDevtools] = useState<ComponentType<{ initialIsOpen?: boolean }> | null>(null);

  useEffect(() => {
    if (!showDevtools) return;

    let cancelled = false;

    import("@tanstack/react-query-devtools")
      .then((mod) => {
        if (!cancelled) {
          setDevtools(() => mod.ReactQueryDevtools);
        }
      })
      .catch((error) => {
        console.warn("Failed to load React Query Devtools", error);
      });

    return () => {
      cancelled = true;
    };
  }, [showDevtools]);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {showDevtools && Devtools ? <Devtools initialIsOpen={false} /> : null}
    </QueryClientProvider>
  );
}
