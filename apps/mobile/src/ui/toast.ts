import { useSyncExternalStore } from "react";

export type ToastTone = "info" | "success" | "error";
export type ToastItem = { id: number; message: string; tone: ToastTone; onPress?: () => void };

let toasts: ToastItem[] = [];
let nextId = 1;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

export function showToast(message: string, tone: ToastTone = "info", onPress?: () => void): void {
  if (toasts.some((t) => t.message === message)) {
    return;
  }
  const item = { id: nextId++, message, tone, onPress };
  toasts = [...toasts.slice(-2), item];
  emit();
  setTimeout(() => dismissToast(item.id), tone === "error" ? 5000 : 3500);
}

export function dismissToast(id: number): void {
  toasts = toasts.filter((t) => t.id !== id);
  emit();
}

export function useToasts(): ToastItem[] {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => toasts,
    () => toasts
  );
}
