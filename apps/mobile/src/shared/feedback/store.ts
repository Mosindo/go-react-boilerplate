import { useSyncExternalStore } from "react";

export type Toast = { id: number; kind: "error" | "info"; message: string };

type FeedbackState = { toasts: Toast[] };

type Listener = () => void;

let state: FeedbackState = { toasts: [] };
let nextId = 1;
const listeners = new Set<Listener>();

function emit() {
  listeners.forEach((listener) => listener());
}

function subscribe(listener: Listener) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function getSnapshot(): FeedbackState {
  return state;
}

function push(kind: Toast["kind"], message: string) {
  if (!message || state.toasts.some((toast) => toast.message === message)) {
    return;
  }
  state = { toasts: [...state.toasts, { id: nextId++, kind, message }].slice(-3) };
  emit();
}

export function showGlobalError(message: string) {
  push("error", message);
}

export function showToast(message: string) {
  push("info", message);
}

export function dismissToast(id: number) {
  state = { toasts: state.toasts.filter((toast) => toast.id !== id) };
  emit();
}

export function resetFeedback() {
  state = { toasts: [] };
  emit();
}

export function useToasts(): Toast[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot).toasts;
}
