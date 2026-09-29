import { useSyncExternalStore } from "react";

export type ToastTone = "success" | "info";
export type Toast = { id: number; message: string; tone: ToastTone };

let toasts: readonly Toast[] = [];
let nextId = 1;
const listeners = new Set<() => void>();
const timers = new Map<number, ReturnType<typeof setTimeout>>();

function emit() {
  listeners.forEach((listener) => listener());
}

export function dismissToast(id: number) {
  const timer = timers.get(id);
  if (timer) {
    clearTimeout(timer);
    timers.delete(id);
  }
  toasts = toasts.filter((toast) => toast.id !== id);
  emit();
}

/** Short-lived confirmation ("Saved", "Photo removed"). Auto-dismisses. */
export function showToast(message: string, tone: ToastTone = "success", durationMs = 3500) {
  if (!message) {
    return;
  }
  const id = nextId++;
  toasts = [...toasts.slice(-2), { id, message, tone }];
  timers.set(
    id,
    setTimeout(() => dismissToast(id), durationMs)
  );
  emit();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function getSnapshot(): readonly Toast[] {
  return toasts;
}

export function useToasts(): readonly Toast[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
