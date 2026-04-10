"use client";

import { WSProvider } from "@multica/core/realtime";
import { useAuthStore } from "./auth";
import { useWorkspaceStore } from "./workspace";
import { webStorage } from "./storage";
import { toast } from "sonner";

function isLocalHostname(hostname: string) {
  return (
    hostname === "localhost" ||
    hostname === "127.0.0.1" ||
    hostname.endsWith(".localhost")
  );
}

function resolveWsUrl() {
  const configured = process.env.NEXT_PUBLIC_WS_URL?.trim();
  if (typeof window === "undefined") {
    return configured || "ws://localhost:8080/ws";
  }

  if (!configured) {
    return "ws://localhost:8080/ws";
  }

  try {
    const url = new URL(configured);
    if (
      isLocalHostname(window.location.hostname) &&
      !isLocalHostname(url.hostname)
    ) {
      return "ws://localhost:8080/ws";
    }
  } catch {
    return "ws://localhost:8080/ws";
  }

  return configured;
}

const WS_URL = resolveWsUrl();

export function WebWSProvider({ children }: { children: React.ReactNode }) {
  return (
    <WSProvider
      wsUrl={WS_URL}
      authStore={useAuthStore}
      workspaceStore={useWorkspaceStore}
      storage={webStorage}
      onToast={(message, type) => {
        if (type === "error") toast.error(message);
        else toast.info(message);
      }}
    >
      {children}
    </WSProvider>
  );
}
