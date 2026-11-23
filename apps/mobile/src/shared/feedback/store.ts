import { useSyncExternalStore } from "react";

type Toast = { id: number; message: string; tone: "default" | "success" | "danger" };

type FeedbackState = {
  error: string | null;
  toast: Toast | null;
};

let state: FeedbackState = { error: null, toast: null };
let toastSeq = 0;
const listeners = new Set<() => void>();

function emit(next: FeedbackState) {
  state = next;
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Global, blocking-free error banner (network / server issues). */
export function showGlobalError(message: string) {
  if (message && state.error !== message) {
    emit({ ...state, error: message });
  }
}

export function clearGlobalError() {
  if (state.error) {
    emit({ ...state, error: null });
  }
}

/** Short confirmation after an action ("Photo ajoutée"). */
export function showToast(message: string, tone: Toast["tone"] = "default") {
  toastSeq += 1;
  emit({ ...state, toast: { id: toastSeq, message, tone } });
}

export function dismissToast(id: number) {
  if (state.toast?.id === id) {
    emit({ ...state, toast: null });
  }
}

export function useGlobalFeedback(): FeedbackState {
  return useSyncExternalStore(
    subscribe,
    () => state,
    () => state
  );
}
