import { useSyncExternalStore } from "react";

export type ToastKind = "error" | "info";
export type ToastState = { id: number; message: string; kind: ToastKind } | null;

let state: ToastState = null;
let counter = 0;
const listeners = new Set<() => void>();

function emit(): void {
  listeners.forEach((listener) => listener());
}

export function showToast(message: string, kind: ToastKind = "info"): void {
  counter += 1;
  state = { id: counter, message, kind };
  emit();
}

export function dismissToast(): void {
  state = null;
  emit();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useToast(): ToastState {
  return useSyncExternalStore(subscribe, () => state);
}
