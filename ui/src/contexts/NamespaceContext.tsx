"use client";

import React, { createContext, useContext, useState, useEffect, ReactNode } from "react";

const NAMESPACE_KEY = "kagent-namespace";

interface NamespaceContextValue {
  namespace: string;
  setNamespace: (ns: string) => void;
}

const NamespaceContext = createContext<NamespaceContextValue | undefined>(undefined);

export function useNamespace(): NamespaceContextValue {
  const ctx = useContext(NamespaceContext);
  if (!ctx) throw new Error("useNamespace must be used within a NamespaceProvider");
  return ctx;
}

export function NamespaceProvider({ children }: { children: ReactNode }) {
  const [namespace, setNamespaceState] = useState<string>("");

  useEffect(() => {
    const stored = localStorage.getItem(NAMESPACE_KEY);
    if (stored) setNamespaceState(stored);
  }, []);

  const setNamespace = (ns: string) => {
    localStorage.setItem(NAMESPACE_KEY, ns);
    setNamespaceState(ns);
  };

  return (
    <NamespaceContext.Provider value={{ namespace, setNamespace }}>
      {children}
    </NamespaceContext.Provider>
  );
}
