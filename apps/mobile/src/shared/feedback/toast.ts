import { useSyncExternalStore } from "react";

export type Toast = { id: number; message: string; tone: "info" | "success" | "error"; onPress?: () => void };

let toasts: Toast[] = [];
let nextId = 1;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

export function showToast(message: string, options: { tone?: Toast["tone"]; onPress?: () => void; durationMs?: number } = {}) {
  const id = nextId++;
  toasts = [...toasts, { id, message, tone: options.tone ?? "info", onPress: options.onPress }];
  emit();
  setTimeout(() => dismissToast(id), options.durationMs ?? 4000);
}

export function dismissToast(id: number) {
  toasts = toasts.filter((toast) => toast.id !== id);
  emit();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useToasts(): Toast[] {
  return useSyncExternalStore(
    subscribe,
    () => toasts,
    () => toasts
  );
}
