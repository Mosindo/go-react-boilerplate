import { useSyncExternalStore } from "react";

export type ToastKind = "success" | "error" | "info";
export type ToastItem = { id: number; kind: ToastKind; message: string };

let items: ToastItem[] = [];
let nextId = 1;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

export function showToast(message: string, kind: ToastKind = "info", durationMs = 3500): number {
  const id = nextId++;
  items = [...items.slice(-2), { id, kind, message }];
  emit();
  if (durationMs > 0) setTimeout(() => dismissToast(id), durationMs);
  return id;
}

export const toast = {
  success: (message: string) => showToast(message, "success"),
  error: (message: string) => showToast(message, "error", 5000),
  info: (message: string) => showToast(message, "info")
};

export function dismissToast(id: number) {
  const next = items.filter((item) => item.id !== id);
  if (next.length !== items.length) {
    items = next;
    emit();
  }
}

export function clearToasts() {
  items = [];
  emit();
}

export function useToasts(): ToastItem[] {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    () => items,
    () => items
  );
}
