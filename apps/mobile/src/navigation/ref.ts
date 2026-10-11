import { createNavigationContainerRef } from '@react-navigation/native';
import type { RootStackParamList } from './types';

/** Lets providers outside screens (match modal, realtime toasts) navigate. */
export const navigationRef = createNavigationContainerRef<RootStackParamList>();
