import { useSyncExternalStore } from "react";

export type ToastKind = "success" | "error" | "info";
export type Toast = { id: number; kind: ToastKind; message: string };

let toasts: Toast[] = [];
let nextId = 1;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

/** Shows a short feedback message; it disappears on its own. */
export function showToast(message: string, kind: ToastKind = "info", durationMs = 4000): void {
  if (toasts.some((t) => t.message === message)) return; // avoid stacking identical messages
  const toast: Toast = { id: nextId++, kind, message };
  toasts = [...toasts.slice(-2), toast];
  emit();
  setTimeout(() => dismissToast(toast.id), durationMs);
}

export const showError = (message: string) => showToast(message, "error", 5000);
export const showSuccess = (message: string) => showToast(message, "success", 3000);

export function dismissToast(id: number): void {
  toasts = toasts.filter((t) => t.id !== id);
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
