import { useNavigation } from '@react-navigation/native';
import type { RootNavigation } from './types';

export function useRootNavigation(): RootNavigation {
  return useNavigation<RootNavigation>();
}
