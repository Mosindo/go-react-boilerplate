import { useCallback } from 'react';
import { errorMessage } from '../api';
import { useToast } from '../ui';

/** Returns a handler that shows a friendly French toast for any thrown error. */
export function useErrorToast(): (error: unknown) => void {
  const toast = useToast();
  return useCallback((error: unknown) => toast.error(errorMessage(error)), [toast]);
}
