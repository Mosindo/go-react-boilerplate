let counter = 0;

/** Client-side temporary id (optimistic messages, toasts). */
export function tempId(prefix = 'tmp'): string {
  counter += 1;
  return `${prefix}-${Date.now().toString(36)}-${counter}`;
}
