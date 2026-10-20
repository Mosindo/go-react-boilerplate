import { useSyncExternalStore } from "react";

type FeedbackState = {
  error: string | null;
  errors: string[];
  loadingCount: number;
  loadingLabel: string | null;
  loadingLabels: string[];
};

type Listener = () => void;

let state: FeedbackState = {
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

export function beginGlobalLoading(label = "Working...") {
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
    errors: nextErrors
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

export function useGlobalFeedback() {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
