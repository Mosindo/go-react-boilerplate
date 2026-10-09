import { useSyncExternalStore } from 'react';

export type ToastKind = 'info' | 'success' | 'error';
export type ToastItem = {
  id: number;
  message: string;
  kind: ToastKind;
  action?: { label: string; onPress: () => void };
};

let items: ToastItem[] = [];
let nextId = 1;
const listeners = new Set<() => void>();
const emit = () => listeners.forEach((l) => l());

export const toast = {
  show(
    message: string,
    options: { kind?: ToastKind; action?: ToastItem['action']; durationMs?: number } = {},
  ): number {
    const id = nextId++;
    items = [
      ...items.slice(-2),
      { id, message, kind: options.kind ?? 'info', action: options.action },
    ];
    emit();
    setTimeout(() => toast.dismiss(id), options.durationMs ?? 4000);
    return id;
  },
  success: (message: string) => toast.show(message, { kind: 'success' }),
  error: (message: string) => toast.show(message, { kind: 'error', durationMs: 5000 }),
  dismiss(id: number) {
    const next = items.filter((t) => t.id !== id);
    if (next.length !== items.length) {
      items = next;
      emit();
    }
  },
  clear() {
    items = [];
    emit();
  },
  getAll: () => items,
};

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

export function useToasts(): ToastItem[] {
  return useSyncExternalStore(subscribe, toast.getAll, toast.getAll);
}
