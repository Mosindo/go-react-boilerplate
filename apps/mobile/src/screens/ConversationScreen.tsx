import React, { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { chatApi, discoverApi, profileApi, safetyApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import type { Message, PublicProfile, ReportReason } from "../api/types";
import { ActionSheet } from "../components/ActionSheet";
import { ReportSheet } from "../components/ReportSheet";
import { useAuth } from "../hooks/useAuth";
import { setActiveConversation, type MessagesPage } from "../hooks/useRealtime";
import type { RootScreenProps } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Avatar, IconButton, Input, MessageItem, Text, radii, spacing, useTheme } from "../shared/ui";
import { formatClock, formatDayLabel } from "../utils/format";

type PendingMessage = { tempId: string; body: string; status: "sending" | "failed" };

type Row = { kind: "message"; message: Message; showDay: boolean } | { kind: "pending"; pending: PendingMessage };

export default function ConversationScreen({ navigation, route }: RootScreenProps<"Conversation">) {
  const { conversationId, userId, user: routeUser } = route.params;
  const { user: me } = useAuth();
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const [text, setText] = useState("");
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [menuOpen, setMenuOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);

  const partnerQuery = useQuery({
    queryKey: queryKeys.publicProfile(userId ?? "none"),
    queryFn: () => profileApi.getPublic(userId as string),
    enabled: !routeUser && !!userId
  });
  const partner: PublicProfile | undefined = routeUser ?? partnerQuery.data;

  const messages = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam, 30),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last: MessagesPage) => (last.hasMore ? last.messages[0]?.id : undefined)
  });

  const all = useMemo<Message[]>(() => {
    const pages = messages.data?.pages ?? [];
    // pages[0] is the newest page; each page is oldest-first
    return [...pages].reverse().flatMap((page) => page.messages);
  }, [messages.data]);

  const unavailable = messages.error instanceof ApiError && messages.error.status === 404;

  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.conversations })
  });
  const lastIncomingId = [...all].reverse().find((m) => m.senderId !== me?.id && !m.readAt)?.id;

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      return () => setActiveConversation(null);
    }, [conversationId])
  );
  useEffect(() => {
    if (lastIncomingId) {
      markRead.mutate();
    }
    // mark read whenever an unread incoming message is on screen
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastIncomingId]);

  const appendToCache = (message: Message) => {
    queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (existing) => {
      if (!existing) {
        return existing;
      }
      if (existing.pages.some((p) => p.messages.some((m) => m.id === message.id))) {
        return existing;
      }
      const [first, ...rest] = existing.pages;
      return first ? { ...existing, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] } : existing;
    });
    void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
  };

  const send = useMutation({
    mutationFn: ({ body }: { tempId: string; body: string }) => chatApi.send(conversationId, body),
    onSuccess: (message, { tempId }) => {
      appendToCache(message);
      setPending((list) => list.filter((p) => p.tempId !== tempId));
    },
    onError: (error, { tempId }) => {
      setPending((list) => list.map((p) => (p.tempId === tempId ? { ...p, status: "failed" } : p)));
      if (error instanceof ApiError && error.status === 404) {
        void messages.refetch();
      }
    }
  });

  const submit = () => {
    const body = text.trim();
    if (!body) {
      return;
    }
    const tempId = `tmp-${Date.now()}`;
    setPending((list) => [...list, { tempId, body, status: "sending" }]);
    setText("");
    send.mutate({ tempId, body });
  };

  const retry = (item: PendingMessage) => {
    setPending((list) => list.map((p) => (p.tempId === item.tempId ? { ...p, status: "sending" } : p)));
    send.mutate({ tempId: item.tempId, body: item.body });
  };

  const goBackToList = () => navigation.goBack();

  const clear = useMutation({
    mutationFn: () => chatApi.clear(conversationId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.messages(conversationId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      showToast("Conversation effacée de votre côté.");
    }
  });
  const unmatch = useMutation({
    mutationFn: async () => {
      const found = (await discoverApi.matches(100)).find((m) => m.conversationId === conversationId);
      if (!found) {
        throw new ApiError(404, "Match introuvable.");
      }
      await discoverApi.unmatch(found.id);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      goBackToList();
    }
  });
  const block = useMutation({
    mutationFn: () => safetyApi.block(partner?.id ?? ""),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      showToast("Personne bloquée.");
      goBackToList();
    }
  });
  const report = useMutation({
    mutationFn: (v: { reason: ReportReason; details: string; block: boolean }) =>
      safetyApi.report(partner?.id ?? "", v.reason, v.details, v.block),
    onSuccess: (_, v) => {
      setReportOpen(false);
      showToast("Merci, votre signalement a été envoyé.");
      if (v.block) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        goBackToList();
      }
    },
    onError: (e) => setReportError(e instanceof ApiError ? e.message : "Envoi impossible.")
  });

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          accessibilityLabel={`Voir le profil de ${partner?.firstName ?? ""}`}
          accessibilityRole="button"
          disabled={!partner}
          onPress={() => partner && navigation.navigate("ProfileDetail", { userId: partner.id, profile: partner })}
          style={styles.titleRow}
        >
          <Avatar name={partner?.firstName} path={partner?.photos[0]?.url} size="sm" />
          <Text variant="heading">{partner?.firstName ?? "Conversation"}</Text>
        </Pressable>
      ),
      headerRight: () => (
        <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenuOpen(true)} size={40} style={styles.noBorder} />
      )
    });
  }, [navigation, partner]);

  const rows = useMemo<Row[]>(() => {
    const chronological: Row[] = all.map((message, i) => ({
      kind: "message" as const,
      message,
      showDay: i === 0 || formatDayLabel(message.createdAt) !== formatDayLabel(all[i - 1]?.createdAt ?? message.createdAt)
    }));
    const queued: Row[] = pending.map((p) => ({ kind: "pending" as const, pending: p }));
    return [...chronological, ...queued].reverse(); // inverted list: newest first
  }, [all, pending]);

  if (messages.isLoading) {
    return <LoadingView fullScreen label="Chargement…" />;
  }
  if (unavailable) {
    return (
      <EmptyView
        actionLabel="Retour"
        icon="heart-dislike-outline"
        message="Cette conversation n'est plus disponible : le match a pris fin."
        onAction={goBackToList}
        title="Conversation fermée"
      />
    );
  }
  if (messages.isError) {
    return <ErrorView message="Impossible de charger les messages." onAction={() => void messages.refetch()} />;
  }

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      style={styles.flex}
    >
      <FlatList
        ListEmptyComponent={
          <View style={styles.emptyWrap}>
            <Text align="center" tone="muted">
              Vous avez matché avec {partner?.firstName ?? "cette personne"}. Dites bonjour 👋
            </Text>
          </View>
        }
        contentContainerStyle={styles.list}
        data={rows}
        inverted={rows.length > 0}
        keyExtractor={(row) => (row.kind === "message" ? row.message.id : row.pending.tempId)}
        onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()}
        onEndReachedThreshold={0.4}
        renderItem={({ item }) => {
          if (item.kind === "pending") {
            return (
              <Pressable disabled={item.pending.status !== "failed"} onPress={() => retry(item.pending)}>
                <MessageItem body={item.pending.body} mine status={item.pending.status} time="" />
              </Pressable>
            );
          }
          const mine = item.message.senderId === me?.id;
          return (
            <View>
              {item.showDay ? (
                <Text align="center" style={styles.day} tone="subtle" variant="caption" weight="semibold">
                  {formatDayLabel(item.message.createdAt)}
                </Text>
              ) : null}
              <MessageItem
                body={item.message.body}
                mine={mine}
                status={mine ? (item.message.readAt ? "read" : "sent") : undefined}
                time={formatClock(item.message.createdAt)}
              />
            </View>
          );
        }}
      />

      <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>
        <Input
          accessibilityLabel="Votre message"
          testID="chat-message-input"
          maxLength={2000}
          multiline
          onChangeText={setText}
          placeholder="Écrire un message…"
          style={styles.input}
          value={text}
        />
        <IconButton
          testID="chat-send-button"
          background={colors.primary}
          color={colors.primaryForeground}
          disabled={text.trim().length === 0}
          icon="send"
          label="Envoyer"
          onPress={submit}
          size={48}
        />
      </View>

      <ActionSheet
        actions={[
          {
            label: "Voir le profil",
            onPress: () => partner && navigation.navigate("ProfileDetail", { userId: partner.id, profile: partner })
          },
          { label: "Effacer la conversation", onPress: () => clear.mutate() },
          { label: "Se désapparier", destructive: true, onPress: () => unmatch.mutate() },
          {
            label: "Signaler",
            destructive: true,
            onPress: () => {
              setReportError(null);
              setReportOpen(true);
            }
          },
          { label: "Bloquer", destructive: true, onPress: () => block.mutate() }
        ]}
        onClose={() => setMenuOpen(false)}
        title={partner?.firstName}
        visible={menuOpen}
      />
      <ReportSheet
        busy={report.isPending}
        error={reportError}
        name={partner?.firstName ?? "cette personne"}
        onClose={() => setReportOpen(false)}
        onSubmit={(reason, details, alsoBlock) => report.mutate({ reason, details, block: alsoBlock })}
        visible={reportOpen}
      />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md, flexGrow: 1 },
  titleRow: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  noBorder: { borderWidth: 0, backgroundColor: "transparent" },
  day: { marginVertical: spacing.md },
  emptyWrap: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: { flex: 1, maxHeight: 120, minHeight: 48, borderRadius: radii.lg }
});
