import { useSyncExternalStore } from "react";
import { strings } from "../../i18n/strings";

export type ToastTone = "info" | "success" | "error";
export type Toast = { id: number; message: string; tone: ToastTone };

type FeedbackState = {
  toast: Toast | null;
  error: string | null;
  errors: string[];
  loadingCount: number;
  loadingLabel: string | null;
  loadingLabels: string[];
};

type Listener = () => void;

let toastSeq = 0;

let state: FeedbackState = {
  toast: null,
  error: null,
  errors: [],
  loadingCount: 0,
  loadingLabel: null,
  loadingLabels: []
};

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

export function beginGlobalLoading(label: string = strings.common.working) {
  const nextLabels = [...state.loadingLabels, label];
  state = {
    ...state,
    loadingCount: nextLabels.length,
    loadingLabel: nextLabels.at(-1) ?? null,
    loadingLabels: nextLabels
  };
  emit();
}

export function endGlobalLoading() {
  const nextLabels = state.loadingLabels.slice(0, -1);
  state = {
    ...state,
    loadingCount: nextLabels.length,
    loadingLabel: nextLabels.at(-1) ?? null,
    loadingLabels: nextLabels
  };
  emit();
}

export function showGlobalError(message: string) {
  if (!message) {
    return;
  }

  const nextErrors = state.errors.at(-1) === message ? state.errors : [...state.errors, message];
  state = {
    ...state,
    error: nextErrors.at(-1) ?? null,
    errors: nextErrors,
    toast: { id: ++toastSeq, message, tone: "error" }
  };
  emit();
}

export function clearGlobalError() {
  if (state.errors.length === 0) {
    return;
  }

  const nextErrors = state.errors.slice(0, -1);
  state = {
    ...state,
    error: nextErrors.at(-1) ?? null,
    errors: nextErrors
  };
  emit();
}

/** Transient message shown by <ToastHost/>. Newer toasts replace older ones. */
export function showToast(message: string, tone: ToastTone = "info") {
  if (!message) {
    return;
  }
  toastSeq += 1;
  state = { ...state, toast: { id: toastSeq, message, tone } };
  emit();
}

export function dismissToast() {
  if (!state.toast) {
    return;
  }
  state = { ...state, toast: null };
  emit();
}

/** Shows any thrown value as an error toast (uses Error.message). */
export function reportError(error: unknown, fallback: string = strings.errors.generic) {
  showToast(error instanceof Error && error.message ? error.message : fallback, "error");
}

/** Drops all queued state; used when signing out and in tests. */
export function resetFeedback() {
  state = { toast: null, error: null, errors: [], loadingCount: 0, loadingLabel: null, loadingLabels: [] };
  emit();
}

export function useGlobalFeedback() {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
