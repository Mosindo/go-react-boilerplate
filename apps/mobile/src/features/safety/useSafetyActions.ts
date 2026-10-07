import { useCallback, useState } from "react";
import { Alert } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { blockUser, reportUser } from "../../api/safety";
import { fetchMatches, unmatch } from "../../api/discover";
import { removeConversation, removeMatch, removeMatchesForUser } from "../../realtime/events";
import { errorMessage } from "../errors";
import { emitSwipeOutcome } from "../discover/swipeBus";
import { queryKeys } from "../queryKeys";
import type { Conversation, Match, ReportReason } from "../types";

type Options = {
  userId: string;
  name: string;
  /** Called after a successful block / unmatch / report so the screen can leave. */
  onDone: () => void;
};

export function useSafetyActions({ userId, name, onDone }: Options) {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [reportVisible, setReportVisible] = useState(false);

  const dropFromCaches = useCallback(
    (conversationIds: string[]) => {
      queryClient.setQueryData<Match[]>(queryKeys.matches, (cur) =>
        cur ? removeMatchesForUser(cur, userId) : cur
      );
      queryClient.setQueryData<Conversation[]>(queryKeys.conversations, (cur) => {
        if (!cur) {
          return cur;
        }
        const ids = new Set([
          ...conversationIds,
          ...cur.filter((c) => c.user.userId === userId).map((c) => c.id)
        ]);
        return cur.filter((c) => !ids.has(c.id));
      });
      for (const id of conversationIds) {
        queryClient.removeQueries({ queryKey: queryKeys.messages(id) });
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      emitSwipeOutcome({ userId, action: "pass", matched: false, match: null });
    },
    [queryClient, userId]
  );

  const runBlock = useCallback(async () => {
    setBusy(true);
    try {
      await blockUser(userId);
      const conversationIds = (queryClient.getQueryData<Conversation[]>(queryKeys.conversations) ?? [])
        .filter((c) => c.user.userId === userId)
        .map((c) => c.id);
      dropFromCaches(conversationIds);
      onDone();
    } catch (err) {
      Alert.alert("Couldn't block", errorMessage(err, "Please try again."));
    } finally {
      setBusy(false);
    }
  }, [dropFromCaches, onDone, queryClient, userId]);

  const confirmBlock = useCallback(() => {
    Alert.alert(
      `Block ${name}?`,
      "They won't be able to see your profile or message you, and any match will be removed.",
      [
        { text: "Cancel", style: "cancel" },
        { text: "Block", style: "destructive", onPress: () => void runBlock() }
      ]
    );
  }, [name, runBlock]);

  const findMatch = useCallback(async (): Promise<{
    matchId: string;
    conversationId: string | null;
  } | null> => {
    const conversation = (queryClient.getQueryData<Conversation[]>(queryKeys.conversations) ?? []).find(
      (c) => c.user.userId === userId
    );
    if (conversation) {
      return { matchId: conversation.matchId, conversationId: conversation.id };
    }
    const matches = await queryClient.fetchQuery({
      queryKey: queryKeys.matches,
      queryFn: fetchMatches,
      staleTime: 10_000
    });
    const match = matches.find((m) => m.user.userId === userId);
    return match ? { matchId: match.id, conversationId: match.conversationId } : null;
  }, [queryClient, userId]);

  const runUnmatch = useCallback(async () => {
    setBusy(true);
    try {
      const found = await findMatch();
      if (!found) {
        Alert.alert("Not matched", `You are no longer matched with ${name}.`);
        return;
      }
      await unmatch(found.matchId);
      queryClient.setQueryData<Match[]>(queryKeys.matches, (cur) =>
        cur ? removeMatch(cur, found.matchId) : cur
      );
      if (found.conversationId) {
        const conversationId = found.conversationId;
        queryClient.setQueryData<Conversation[]>(queryKeys.conversations, (cur) =>
          cur ? removeConversation(cur, conversationId) : cur
        );
      }
      dropFromCaches(found.conversationId ? [found.conversationId] : []);
      onDone();
    } catch (err) {
      Alert.alert("Couldn't unmatch", errorMessage(err, "Please try again."));
    } finally {
      setBusy(false);
    }
  }, [dropFromCaches, findMatch, name, onDone, queryClient]);

  const confirmUnmatch = useCallback(() => {
    Alert.alert(
      `Unmatch with ${name}?`,
      "Your conversation will be deleted and you won't see each other again.",
      [
        { text: "Cancel", style: "cancel" },
        { text: "Unmatch", style: "destructive", onPress: () => void runUnmatch() }
      ]
    );
  }, [name, runUnmatch]);

  const submitReport = useCallback(
    async (reason: ReportReason, details: string) => {
      setBusy(true);
      try {
        await reportUser(userId, reason, details);
        setReportVisible(false);
        Alert.alert("Report sent", `Thanks for letting us know. Would you also like to block ${name}?`, [
          { text: "Not now", style: "cancel", onPress: onDone },
          { text: "Block", style: "destructive", onPress: () => void runBlock() }
        ]);
      } catch (err) {
        Alert.alert("Couldn't send report", errorMessage(err, "Please try again."));
      } finally {
        setBusy(false);
      }
    },
    [name, onDone, runBlock, userId]
  );

  return {
    busy,
    reportVisible,
    openReport: useCallback(() => setReportVisible(true), []),
    closeReport: useCallback(() => setReportVisible(false), []),
    confirmBlock,
    confirmUnmatch,
    submitReport
  };
}
