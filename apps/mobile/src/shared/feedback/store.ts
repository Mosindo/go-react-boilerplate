import { useSyncExternalStore } from "react";

export type Toast = { id: number; message: string; tone: "info" | "success" | "danger" };

let toasts: Toast[] = [];
let nextId = 1;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Shows a transient message at the bottom of the screen (auto-dismissed by <ToastHost />). */
export function showToast(message: string, tone: Toast["tone"] = "info"): void {
  if (!message) {
    return;
  }
  toasts = [...toasts.filter((toast) => toast.message !== message), { id: nextId++, message, tone }].slice(-3);
  emit();
}

export const showGlobalError = (message: string) => showToast(message, "danger");

export function dismissToast(id: number): void {
  toasts = toasts.filter((toast) => toast.id !== id);
  emit();
}

export function useToasts(): Toast[] {
  return useSyncExternalStore(
    subscribe,
    () => toasts,
    () => toasts
  );
}
