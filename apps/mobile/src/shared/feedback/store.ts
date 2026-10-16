import { useSyncExternalStore } from "react";

type FeedbackState = {
  error: string | null;
  errors: string[];
};

type Listener = () => void;

let state: FeedbackState = {
  error: null,
  errors: []
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
