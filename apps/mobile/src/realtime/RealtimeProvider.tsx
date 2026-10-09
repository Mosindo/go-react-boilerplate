import { useEffect } from 'react';
import { AppState } from 'react-native';
import { useQueryClient } from '@tanstack/react-query';
import { notificationsApi } from '../api/notifications';
import type { Conversation, RealtimeEvent } from '../api/types';
import { toWebSocketUrl, API_BASE_URL } from '../config';
import { fr } from '../i18n/fr';
import { leaveChat, openChat, openTab } from '../navigation/ref';
import { qk } from '../query/keys';
import { toast } from '../shared/toastStore';
import { activeConversation, matchCelebration } from '../store/appState';
import { createStore } from '../store/simpleStore';
import { applyRealtimeEvent, refetchAfterReconnect } from './cacheUpdaters';
import { RealtimeClient, type RealtimeStatus } from './client';

export const realtimeStatus = createStore<RealtimeStatus>('idle');

/** Owns the WebSocket for the signed-in user (mount it once, below the QueryClientProvider). */
export function RealtimeProvider({ userId }: { userId: string }) {
  const qc = useQueryClient();

  useEffect(() => {
    const handleEvent = (event: RealtimeEvent) => {
      const effect = applyRealtimeEvent(qc, event, {
        myUserId: userId,
        activeConversationId: activeConversation.get(),
      });
      if (!effect) return;
      if (effect.kind === 'conversationRemoved') {
        leaveChat(effect.conversationId);
        return;
      }
      const { conversationId, user } = effect;
      // The swipe that created this match already shows the celebration modal.
      if (conversationId && matchCelebration.get()?.conversationId === conversationId) return;
      toast.show(effect.message, {
        action: conversationId
          ? {
              label: fr.match.open,
              onPress: () => {
                const known = qc
                  .getQueryData<Conversation[]>(qk.conversations)
                  ?.find((c) => c.id === conversationId);
                const target = known?.user ?? user ?? null;
                if (target) openChat(conversationId, target);
                else openTab('Messages');
              },
            }
          : undefined,
      });
    };

    const client = new RealtimeClient({
      getTicket: notificationsApi.wsTicket,
      wsBaseUrl: toWebSocketUrl(API_BASE_URL),
      onEvent: handleEvent,
      onOpen: (isReconnect) => {
        if (isReconnect) refetchAfterReconnect(qc);
      },
      onStatus: (s) => realtimeStatus.set(s),
    });
    client.start();

    const sub = AppState.addEventListener('change', (state) => {
      if (state !== 'active') return;
      if (client.isOpen()) {
        refetchAfterReconnect(qc);
      } else {
        client.reconnectNow();
      }
    });

    return () => {
      sub.remove();
      client.stop();
      realtimeStatus.set('idle');
    };
  }, [qc, userId]);

  return null;
}
