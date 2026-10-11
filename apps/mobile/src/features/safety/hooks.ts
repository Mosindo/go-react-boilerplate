import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { discoverApi, queryKeys, safetyApi } from '../../api';
import type { PublicProfile } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { conversationsWithout } from '../../realtime/cache';
import type { ConversationsData } from '../../realtime/cache';
import { useToast } from '../../ui';
import { useDeck } from '../discover/DeckContext';

export function useBlocks() {
  return useQuery({ queryKey: queryKeys.blocks, queryFn: () => safetyApi.blocks() });
}

/** Removes everything tied to a conversation from the caches. */
function useForgetConversation() {
  const qc = useQueryClient();
  return (conversationId: string) => {
    qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
      old ? conversationsWithout(old, conversationId) : old,
    );
    qc.removeQueries({ queryKey: queryKeys.messages(conversationId) });
    void qc.invalidateQueries({ queryKey: queryKeys.conversations });
  };
}

export interface SafetyTarget {
  userId: string;
  firstName: string;
  /** Present when the two users are matched. */
  matchId?: string;
  conversationId?: string;
}

export function useBlockUser(onDone?: () => void) {
  const qc = useQueryClient();
  const toast = useToast();
  const onError = useErrorToast();
  const deck = useDeck();
  const forget = useForgetConversation();
  return useMutation({
    mutationFn: (target: SafetyTarget) => safetyApi.block(target.userId),
    onSuccess: (_data, target) => {
      deck.remove(target.userId);
      if (target.conversationId) forget(target.conversationId);
      qc.setQueryData<PublicProfile[]>(queryKeys.likes, (old) =>
        old?.filter((p) => p.userId !== target.userId),
      );
      void qc.invalidateQueries({ queryKey: queryKeys.blocks });
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      toast.success(fr.safety.blocked(target.firstName));
      onDone?.();
    },
    onError,
  });
}

export function useUnblockUser() {
  const qc = useQueryClient();
  const toast = useToast();
  const onError = useErrorToast();
  return useMutation({
    mutationFn: (userId: string) => safetyApi.unblock(userId),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.blocks });
      toast.success(fr.safety.unblocked);
    },
    onError,
  });
}

export function useUnmatch(onDone?: () => void) {
  const toast = useToast();
  const onError = useErrorToast();
  const forget = useForgetConversation();
  return useMutation({
    mutationFn: (target: SafetyTarget) => {
      if (!target.matchId) throw new Error('missing_match');
      return discoverApi.unmatch(target.matchId);
    },
    onSuccess: (_data, target) => {
      if (target.conversationId) forget(target.conversationId);
      toast.success(fr.safety.unmatched(target.firstName));
      onDone?.();
    },
    onError,
  });
}
