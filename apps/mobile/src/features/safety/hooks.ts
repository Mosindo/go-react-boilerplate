import { useMutation, useQueryClient } from "@tanstack/react-query";
import { matchesApi, queryKeys, safetyApi, type ConversationItem, type MatchItem, type ReportReason } from "../../api";
import { removeConversation, removeMatch, removeMatchByConversation, type Paged } from "../../realtime/cache";
import { showToast } from "../../shared/feedback";

/** Drops every cached trace of a person (match, conversation, profile) after unmatch or block. */
function useForgetPerson() {
  const queryClient = useQueryClient();
  return (userId: string, conversationId?: string) => {
    if (conversationId) {
      queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) =>
        removeConversation(data, conversationId)
      );
      queryClient.setQueryData<Paged<MatchItem>>(queryKeys.matches, (data) =>
        removeMatchByConversation(data, conversationId)
      );
      queryClient.removeQueries({ queryKey: queryKeys.messages(conversationId) });
    }
    queryClient.removeQueries({ queryKey: queryKeys.publicProfile(userId) });
    void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    void queryClient.invalidateQueries({ queryKey: queryKeys.unreadCount });
    void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  };
}

export function useBlockUser(onDone: () => void) {
  const queryClient = useQueryClient();
  const forget = useForgetPerson();
  return useMutation({
    mutationFn: ({ userId }: { userId: string; conversationId?: string }) => safetyApi.block(userId),
    onSuccess: (_void, { userId, conversationId }) => {
      forget(userId, conversationId);
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      showToast("Utilisateur bloqué.");
      onDone();
    }
  });
}

export function useUnmatch(onDone: () => void) {
  const queryClient = useQueryClient();
  const forget = useForgetPerson();
  return useMutation({
    mutationFn: ({ matchId }: { matchId: string; userId: string; conversationId: string }) =>
      matchesApi.unmatch(matchId),
    onSuccess: (_void, { matchId, userId, conversationId }) => {
      queryClient.setQueryData<Paged<MatchItem>>(queryKeys.matches, (data) => removeMatch(data, matchId));
      forget(userId, conversationId);
      showToast("Match supprimé.");
      onDone();
    }
  });
}

export function useReportUser(onDone: () => void) {
  return useMutation({
    mutationFn: (input: { userId: string; reason: ReportReason; details?: string }) => safetyApi.report(input),
    onSuccess: () => {
      showToast("Merci, votre signalement a bien été envoyé.");
      onDone();
    }
  });
}
