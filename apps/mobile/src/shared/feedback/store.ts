import { useSyncExternalStore } from "react";

export type ToastTone = "info" | "error" | "success";
export type Toast = { id: number; message: string; tone: ToastTone };

let toasts: Toast[] = [];
let nextId = 1;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

/** Shows a short message; identical consecutive messages are collapsed. */
export function showToast(message: string, tone: ToastTone = "info"): void {
  if (!message || toasts.at(-1)?.message === message) return;
  toasts = [...toasts.slice(-2), { id: nextId++, message, tone }];
  emit();
}

export function showGlobalError(message: string): void {
  showToast(message, "error");
}

export function dismissToast(id: number): void {
  toasts = toasts.filter((toast) => toast.id !== id);
  emit();
}

export function resetToasts(): void {
  toasts = [];
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
