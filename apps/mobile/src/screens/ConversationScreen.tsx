import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useFocusEffect, useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { fetchMessages, hideConversation, markConversationRead, sendMessage, unmatch, type MessagePage } from "../api/platform";
import { errorMessage, isApiError } from "../api/client";
import type { ChatMessage } from "../api/types";
import { SafetyMenu } from "../features/safety/SafetyMenu";
import { useConversations, useMatches } from "../hooks/queries";
import { appendMessageToCache, useRealtime } from "../hooks/useRealtime";
import { useAuth } from "../hooks/useAuth";
import { formatDayLabel, sameDay } from "../lib/format";
import { queryKeys } from "../lib/queryClient";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { ActionSheet, Avatar, ConfirmDialog, MessageItem, Notice, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;
type Route = RouteProp<RootStackParamList, "Conversation">;

type Row = { kind: "message"; key: string; message: ChatMessage; pending?: boolean } | { kind: "day"; key: string; label: string };

export default function ConversationScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const { params } = useRoute<Route>();
  const { conversationId, userId, name, photoId } = params;
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { connected } = useRealtime();
  const matches = useMatches();
  const conversations = useConversations();
  const [draft, setDraft] = useState("");
  const [sendError, setSendError] = useState<string | null>(null);
  const [pending, setPending] = useState<{ id: string; body: string; createdAt: string; afterId: number }[]>([]);
  const [menu, setMenu] = useState<"actions" | "safety" | "unmatch" | null>(null);
  const listRef = useRef<FlatList<Row>>(null);

  const query = useInfiniteQuery<MessagePage, Error, InfiniteData<MessagePage, number | null>, readonly string[], number | null>({
    queryKey: queryKeys.messages(conversationId),
    initialPageParam: null,
    queryFn: ({ pageParam }) => fetchMessages(conversationId, pageParam),
    getNextPageParam: (last) => last.nextBefore,
    refetchInterval: connected ? false : 5000
  });

  const messages = useMemo(() => {
    // Pages are newest-first; each page is chronological. Flatten to oldest -> newest.
    const pages = [...(query.data?.pages ?? [])].reverse();
    return pages.flatMap((p) => p.messages);
  }, [query.data]);

  const matchId =
    matches.data?.find((m) => m.conversationId === conversationId)?.id ?? conversations.data?.find((c) => c.id === conversationId)?.matchId;

  const markRead = useCallback(() => {
    markConversationRead(conversationId)
      .then(() =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: queryKeys.conversations }),
          queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
        ])
      )
      .catch(() => undefined);
  }, [conversationId, queryClient]);

  useFocusEffect(
    useCallback(() => {
      markRead();
    }, [markRead])
  );

  const lastIncoming = [...messages].reverse().find((m) => m.senderId !== user?.id)?.id;
  useEffect(() => {
    if (lastIncoming !== undefined) {
      markRead();
    }
  }, [lastIncoming, markRead]);

  const send = useMutation({
    mutationFn: ({ body }: { body: string; tempId: string }) => sendMessage(conversationId, body),
    onSuccess: (message, { tempId }) => {
      appendMessageToCache(queryClient, message);
      setPending((list) => list.filter((p) => p.id !== tempId));
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    },
    onError: (error, { tempId, body }) => {
      setPending((list) => list.filter((p) => p.id !== tempId));
      setDraft((current) => current || body);
      setSendError(isApiError(error, 404) ? "Cette conversation n'est plus disponible." : errorMessage(error, "Message non envoyé."));
    }
  });

  function submit() {
    const body = draft.trim();
    if (!body || send.isPending) {
      return;
    }
    const tempId = `pending-${Date.now()}`;
    setSendError(null);
    setDraft("");
    setPending((list) => [...list, { id: tempId, body, createdAt: new Date().toISOString(), afterId: messages.at(-1)?.id ?? 0 }]);
    send.mutate({ body, tempId });
  }

  const leave = useMutation({
    mutationFn: async (action: "unmatch" | "hide") => {
      if (action === "unmatch" && matchId) {
        await unmatch(matchId);
      } else {
        await hideConversation(conversationId);
      }
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.matches }),
        queryClient.invalidateQueries({ queryKey: queryKeys.conversations })
      ]);
      navigation.goBack();
    },
    onError: (error) => setSendError(errorMessage(error))
  });

  // Inverted list: newest first, so build rows newest -> oldest with day separators.
  const rows = useMemo<Row[]>(() => {
    const all: Row[] = messages.map((message) => ({ kind: "message", key: String(message.id), message }));
    // A pending bubble is dropped as soon as the server copy (pushed over the socket) shows up.
    const unconfirmed = pending.filter((p) => !messages.some((m) => m.id > p.afterId && m.senderId === user?.id && m.body === p.body));
    unconfirmed.forEach((p) =>
      all.push({
        kind: "message",
        key: p.id,
        pending: true,
        message: { id: -1, conversationId, senderId: user?.id ?? "", body: p.body, createdAt: p.createdAt, read: false }
      })
    );
    const out: Row[] = [];
    for (let i = all.length - 1; i >= 0; i -= 1) {
      const row = all[i];
      if (row.kind !== "message") {
        continue;
      }
      out.push(row);
      const older = all[i - 1];
      if (!older || older.kind !== "message" || !sameDay(older.message.createdAt, row.message.createdAt)) {
        out.push({ kind: "day", key: `day-${row.key}`, label: formatDayLabel(row.message.createdAt) });
      }
    }
    return out;
  }, [conversationId, messages, pending, user?.id]);

  const openProfile = () => navigation.navigate("ProfileDetail", { userId });
  const gone = query.isError && isApiError(query.error, 404);

  return (
    <SafeAreaLayout edges={["top", "bottom"]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <View style={[styles.header, { borderColor: colors.border }]}>
          <Pressable
            accessibilityLabel="Retour"
            accessibilityRole="button"
            hitSlop={10}
            onPress={() => navigation.goBack()}
            testID="chat-back"
          >
            <Ionicons color={colors.text} name="chevron-back" size={26} />
          </Pressable>
          <Pressable accessibilityLabel={`Voir le profil de ${name}`} accessibilityRole="button" onPress={openProfile} style={styles.who}>
            <Avatar name={name} photoId={photoId} size="sm" />
            <Text numberOfLines={1} variant="heading" weight="bold">
              {name}
            </Text>
          </Pressable>
          <Pressable
            accessibilityLabel="Plus d'options"
            accessibilityRole="button"
            hitSlop={10}
            onPress={() => setMenu("actions")}
            testID="chat-menu"
          >
            <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
          </Pressable>
        </View>

        {query.isLoading ? (
          <LoadingView />
        ) : gone ? (
          <ErrorView
            actionLabel="Retour"
            message="Cette conversation n'existe plus ou a été fermée."
            onAction={() => navigation.goBack()}
            title="Conversation indisponible"
          />
        ) : query.isError ? (
          <ErrorView message={query.error.message} onAction={() => void query.refetch()} />
        ) : (
          <FlatList
            contentContainerStyle={styles.messages}
            data={rows}
            inverted
            keyExtractor={(row) => row.key}
            ListEmptyComponent={
              <View style={styles.emptyChat}>
                <Text style={styles.center} tone="muted">
                  C'est un match ! Lancez la conversation avec {name} 👋
                </Text>
              </View>
            }
            onEndReached={() => {
              if (query.hasNextPage && !query.isFetchingNextPage) {
                void query.fetchNextPage();
              }
            }}
            ref={listRef}
            renderItem={({ item }) =>
              item.kind === "day" ? (
                <Text style={styles.day} tone="subtle" variant="caption" weight="semibold">
                  {item.label}
                </Text>
              ) : (
                <MessageItem
                  body={item.message.body}
                  createdAt={item.message.createdAt}
                  mine={item.message.senderId === user?.id}
                  pending={item.pending}
                  read={item.message.read}
                />
              )
            }
            testID="message-list"
          />
        )}

        {sendError ? <Notice message={sendError} style={styles.notice} tone="danger" /> : null}
        {!gone ? (
          <View style={[styles.composer, { borderColor: colors.border, backgroundColor: colors.background }]}>
            <TextInput
              accessibilityLabel="Votre message"
              blurOnSubmit={false}
              maxLength={2000}
              multiline
              onChangeText={setDraft}
              onKeyPress={(event) => {
                // Web: Enter sends, Shift+Enter inserts a new line.
                const native = event.nativeEvent as unknown as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                  event.preventDefault();
                  submit();
                }
              }}
              placeholder="Écrivez un message…"
              placeholderTextColor={colors.textSubtle}
              style={[styles.input, { backgroundColor: colors.surface, borderColor: colors.border, color: colors.text }]}
              testID="chat-input"
              value={draft}
            />
            <Pressable
              accessibilityLabel="Envoyer"
              accessibilityRole="button"
              accessibilityState={{ disabled: !draft.trim() }}
              disabled={!draft.trim()}
              onPress={submit}
              style={[styles.send, { backgroundColor: colors.primary, opacity: draft.trim() ? 1 : 0.4 }]}
              testID="chat-send"
            >
              <Ionicons color={colors.primaryForeground} name="arrow-up" size={22} />
            </Pressable>
          </View>
        ) : null}
      </KeyboardAvoidingView>

      <ActionSheet
        actions={[
          { label: `Voir le profil de ${name}`, onPress: openProfile },
          { label: "Signaler ou bloquer", onPress: () => setMenu("safety") },
          { label: "Supprimer la conversation", onPress: () => leave.mutate("hide") },
          { label: "Annuler le match", destructive: true, onPress: () => setMenu("unmatch"), disabled: !matchId }
        ]}
        onClose={() => setMenu((current) => (current === "actions" ? null : current))}
        visible={menu === "actions"}
      />
      <SafetyMenu
        name={name}
        onBlocked={() => navigation.goBack()}
        onClose={() => setMenu(null)}
        userId={userId}
        visible={menu === "safety"}
      />
      <ConfirmDialog
        confirmLabel="Annuler le match"
        destructive
        loading={leave.isPending}
        message="La conversation sera fermée pour vous deux. Vous ne pourrez plus vous retrouver dans la découverte."
        onCancel={() => setMenu(null)}
        onConfirm={() => leave.mutate("unmatch")}
        title={`Annuler le match avec ${name} ?`}
        visible={menu === "unmatch"}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  who: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  messages: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    gap: spacing.sm,
    width: "100%",
    maxWidth: 640,
    alignSelf: "center"
  },
  day: { textAlign: "center", paddingVertical: spacing.sm },
  emptyChat: { padding: spacing.xl, transform: [{ scaleY: -1 }] },
  center: { textAlign: "center" },
  notice: { marginHorizontal: spacing.lg },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
    width: "100%",
    maxWidth: 640,
    alignSelf: "center"
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderWidth: 1.5,
    borderRadius: radii.xl,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    fontSize: 16
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
