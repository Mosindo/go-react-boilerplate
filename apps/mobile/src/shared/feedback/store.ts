import { useSyncExternalStore } from "react";

export type ToastTone = "info" | "success" | "danger";

export type Toast = {
  id: number;
  message: string;
  tone: ToastTone;
  actionLabel?: string;
  onAction?: () => void;
};

type ToastInput = Omit<Toast, "id" | "tone"> & { tone?: ToastTone };

let toasts: Toast[] = [];
let nextId = 1;
const listeners = new Set<() => void>();
const timers = new Map<number, ReturnType<typeof setTimeout>>();

function emit(): void {
  listeners.forEach((listener) => listener());
}

export function dismissToast(id: number): void {
  const timer = timers.get(id);
  if (timer) {
    clearTimeout(timer);
    timers.delete(id);
  }
  toasts = toasts.filter((toast) => toast.id !== id);
  emit();
}

/** Shows a short banner. The latest toast wins so messages never pile up. */
export function showToast(input: ToastInput, durationMs = 4500): number {
  const id = nextId++;
  toasts.forEach((toast) => dismissToast(toast.id));
  toasts = [{ tone: "info", ...input, id }];
  timers.set(
    id,
    setTimeout(() => dismissToast(id), durationMs)
  );
  emit();
  return id;
}

export const showError = (message: string) => showToast({ message, tone: "danger" });
export const showSuccess = (message: string) => showToast({ message, tone: "success" }, 3000);

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useToasts(): Toast[] {
  return useSyncExternalStore(
    subscribe,
    () => toasts,
    () => toasts
  );
}
