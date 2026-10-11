import { useEffect } from 'react';
import type { ReactNode } from 'react';
import { AppState } from 'react-native';
import { useQueryClient } from '@tanstack/react-query';
import type { NavigationContainerRefWithCurrent } from '@react-navigation/native';
import { API_BASE_URL, queryKeys, realtimeApi } from '../api';
import { toWsUrl } from '../api/baseUrl';
import { useAuth } from '../features/auth/AuthContext';
import { fr } from '../i18n';
import type { RootStackParamList } from '../navigation/types';
import { useToast } from '../ui';
import { getActiveConversation } from './activeConversation';
import { applyRealtimeEvent, refetchAfterReconnect } from './cache';
import type { ConversationsData } from './cache';
import { RealtimeClient } from './client';
import { openWebSocket } from './socket';
import { isOwnSwipeMatch } from './swipeGuard';

/** Keeps one WebSocket open while signed in and funnels events into the query caches. */
export function RealtimeProvider({
  children,
  navigationRef,
}: {
  children: ReactNode;
  navigationRef: NavigationContainerRefWithCurrent<RootStackParamList>;
}) {
  const qc = useQueryClient();
  const toast = useToast();
  const { user } = useAuth();
  const userId = user?.id;

  useEffect(() => {
    if (!userId) return;

    const client = new RealtimeClient({
      getUrl: async () => toWsUrl(API_BASE_URL, (await realtimeApi.ticket()).ticket),
      openSocket: openWebSocket,
      onReconnected: () => refetchAfterReconnect(qc),
      onEvent: (event) => {
        const effects = applyRealtimeEvent(qc, event, {
          myUserId: userId,
          activeConversationId: getActiveConversation(),
        });
        const conversations = () =>
          qc
            .getQueryData<ConversationsData>(queryKeys.conversations)
            ?.pages.flatMap((p) => p.conversations) ?? [];

        if (effects.newMatch) {
          const conversation = effects.newMatch;
          // The person who swiped last already sees the match modal.
          if (!isOwnSwipeMatch(conversation.user.userId)) {
            toast.show(fr.realtime.newMatch(conversation.user.firstName), 'success', {
              label: fr.realtime.open,
              onPress: () => {
                if (navigationRef.isReady()) navigationRef.navigate('Chat', { conversation });
              },
            });
          }
        } else if (effects.newNotification?.type === 'new_like') {
          toast.show(effects.newNotification.title, 'info');
        }
        if (effects.incomingMessage) {
          const message = effects.incomingMessage;
          const conversation = conversations().find((c) => c.id === message.conversationId);
          if (conversation) {
            toast.show(fr.realtime.newMessage(conversation.user.firstName), 'info', {
              label: fr.realtime.open,
              onPress: () => {
                if (navigationRef.isReady()) navigationRef.navigate('Chat', { conversation });
              },
            });
          }
        }
        if (effects.removedConversationId && navigationRef.isReady()) {
          const route = navigationRef.getCurrentRoute();
          const params = route?.params as { conversation?: { id?: string } } | undefined;
          if (
            route?.name === 'Chat' &&
            params?.conversation?.id === effects.removedConversationId
          ) {
            navigationRef.goBack();
            toast.show(fr.realtime.conversationRemoved, 'info');
          }
        }
      },
    });
    client.start();

    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active') client.ensureConnected();
    });
    return () => {
      subscription.remove();
      client.stop();
    };
  }, [userId, qc, toast, navigationRef]);

  return <>{children}</>;
}
