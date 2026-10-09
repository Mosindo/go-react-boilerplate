import { createNavigationContainerRef } from '@react-navigation/native';
import type { ChatUser, MainStackParamList } from './types';

export const navigationRef = createNavigationContainerRef<MainStackParamList>();

export function openChat(conversationId: string, user: ChatUser): void {
  if (navigationRef.isReady()) navigationRef.navigate('Chat', { conversationId, user });
}

export function openTab(tab: 'Discover' | 'Likes' | 'Messages' | 'Profile'): void {
  if (navigationRef.isReady()) navigationRef.navigate('Tabs', { screen: tab });
}

/** Leave the chat screen of a conversation that disappeared (unmatch, block). */
export function leaveChat(conversationId: string): void {
  if (!navigationRef.isReady()) return;
  const route = navigationRef.getCurrentRoute();
  const params = route?.params as { conversationId?: string } | undefined;
  if (
    route?.name === 'Chat' &&
    params?.conversationId === conversationId &&
    navigationRef.canGoBack()
  ) {
    navigationRef.goBack();
  }
}
