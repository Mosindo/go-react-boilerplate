import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { errorMessage } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import type { Message, MessagesPage } from "../../api/types";
import { SafetySheet } from "../../components/SafetySheet";
import { useAuth } from "../../hooks/useAuth";
import { keys, useMatches } from "../../hooks/queries";
import { useRealtimeStatus } from "../../hooks/realtimeStatus";
import type { AppStackParamList } from "../../navigation/types";
import { radii, spacing, type as typeScale, useTheme } from "../../theme";
import { Avatar, EmptyState, ErrorState, Icon, IconButton, Loading, Text, showToast } from "../../ui";
import { dayLabel, messageTime } from "../../utils/format";

type PendingMessage = Message & { savedId?: string };

/**
 * Returns optimistic messages that have no server copy yet. A server message
 * confirms a pending one if it has its saved id, or (when the socket event beat
 * the HTTP response) the same sender and content and was created after it was queued.
 * Each server message confirms at most one pending entry.
 */
function unconfirmed(pending: PendingMessage[], server: Message[]): PendingMessage[] {
  const claimed = new Set<string>();
  return pending.filter((p) => {
    const match = server.find(
      (m) =>
        !claimed.has(m.id) &&
        (m.id === p.savedId ||
          (m.senderUserId === p.senderUserId && m.content === p.content && new Date(m.createdAt).getTime() >= new Date(p.createdAt).getTime() - 30_000))
    );
    if (match) claimed.add(match.id);
    return !match;
  });
}

type Row = { kind: "message"; message: Message; pending?: boolean } | { kind: "day"; id: string; label: string };

export default function ChatScreen({ route, navigation }: NativeStackScreenProps<AppStackParamList, "Chat">) {
  const { userId, firstName } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const qc = useQueryClient();
  const connected = useRealtimeStatus();
  const matches = useMatches();
  const match = matches.data?.find((m) => m.user.userId === userId);
  const [draft, setDraft] = useState("");
  // Optimistic messages; `savedId` is set once the server confirmed them so the
  // confirmed copy (from the cache or the socket) replaces them without a duplicate frame.
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [sheet, setSheet] = useState(false);

  const query = useInfiniteQuery({
    queryKey: keys.messages(userId),
    queryFn: ({ pageParam }) => chatApi.messages(userId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.hasMore ? last.messages[0]?.id : undefined),
    refetchInterval: connected ? false : 8_000
  });

  const all = useMemo(() => {
    const pages = query.data?.pages ?? [];
    const list: Message[] = [];
    for (let i = pages.length - 1; i >= 0; i--) list.push(...(pages[i]?.messages ?? []));
    return list;
  }, [query.data]);
  const otherLastReadAt = query.data?.pages[0]?.otherLastReadAt;
  const lastIncomingId = [...all].reverse().find((m) => m.senderUserId === userId)?.id;

  // Mark read when the screen is focused and whenever a new incoming message lands.
  const markRead = useCallback(() => {
    chatApi
      .markRead(userId)
      .then(() => {
        void qc.invalidateQueries({ queryKey: keys.chats });
        void qc.invalidateQueries({ queryKey: keys.unreadChats });
      })
      .catch(() => undefined);
  }, [qc, userId]);
  useFocusEffect(markRead);
  useEffect(() => {
    if (lastIncomingId) markRead();
  }, [lastIncomingId, markRead]);

  const send = async () => {
    const content = draft.trim();
    if (!content || !user) return;
    const temp: Message = {
      id: `pending-${Date.now()}`,
      conversationId: "",
      senderUserId: user.id,
      recipientUserId: userId,
      content,
      createdAt: new Date().toISOString()
    };
    setDraft("");
    setPending((p) => [...p, temp]);
    let delivered = false;
    try {
      const saved = await chatApi.send(userId, content);
      delivered = true;
      setPending((p) => p.map((m) => (m.id === temp.id ? { ...m, savedId: saved.id } : m)));
      qc.setQueryData<InfiniteData<MessagesPage>>(keys.messages(userId), (data) => {
        if (!data) return data;
        const [first, ...rest] = data.pages;
        if (!first || first.messages.some((m) => m.id === saved.id)) return data;
        return { ...data, pages: [{ ...first, messages: [...first.messages, saved] }, ...rest] };
      });
      void qc.invalidateQueries({ queryKey: keys.chats });
      void qc.invalidateQueries({ queryKey: keys.matches });
    } catch (e) {
      setDraft(content);
      showToast(errorMessage(e, "Message non envoyé."), "error");
    } finally {
      if (!delivered) setPending((p) => p.filter((m) => m.id !== temp.id));
    }
  };

  // Optimistic entries whose server copy is already listed (via HTTP or the socket, whichever came first).
  const visiblePending = useMemo(() => unconfirmed(pending, all), [pending, all]);

  // Drop confirmed optimistic entries once the request settled.
  useEffect(() => {
    setPending((p) => {
      const keep = unconfirmed(p, all);
      return keep.length === p.length ? p : p.filter((m) => !m.savedId || keep.includes(m));
    });
  }, [all]);

  // Inverted list: newest first, with day separators.
  const rows = useMemo<Row[]>(() => {
    const out: Row[] = [];
    const combined = [...all, ...visiblePending];
    for (let i = combined.length - 1; i >= 0; i--) {
      const m = combined[i] as Message;
      out.push({ kind: "message", message: m, pending: m.id.startsWith("pending-") });
      const prev = combined[i - 1];
      const day = dayLabel(m.createdAt);
      if (!prev || dayLabel(prev.createdAt) !== day) out.push({ kind: "day", id: `day-${m.id}`, label: day });
    }
    return out;
  }, [all, visiblePending]);

  const lastSeenOwnId = useMemo(() => {
    if (!otherLastReadAt) return undefined;
    const read = new Date(otherLastReadAt).getTime();
    return [...all].reverse().find((m) => m.senderUserId === user?.id && new Date(m.createdAt).getTime() <= read)?.id;
  }, [all, otherLastReadAt, user?.id]);

  const photo = match?.user.photos[0]?.url;

  return (
    <SafeAreaView edges={["top", "bottom"]} style={[styles.fill, { backgroundColor: colors.background }]}>
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${firstName}`}
          style={styles.headerUser}
          onPress={() => navigation.navigate("ProfileView", { userId, fromChat: true })}
        >
          <Avatar url={photo} size={38} />
          <Text variant="heading" numberOfLines={1}>
            {firstName}
          </Text>
        </Pressable>
        <IconButton icon="ellipsis-horizontal" label="Options de sécurité" onPress={() => setSheet(true)} />
      </View>
      <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {query.isLoading ? (
          <Loading />
        ) : query.error ? (
          <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState icon="chatbubbles-outline" title={`Dites bonjour à ${firstName}`} message="Une question sur son profil est souvent le meilleur début." />
        ) : (
          <FlatList<Row>
            inverted
            data={rows}
            keyExtractor={(r) => (r.kind === "day" ? r.id : r.message.id)}
            onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
            onEndReachedThreshold={0.3}
            contentContainerStyle={styles.list}
            renderItem={({ item }) => {
              if (item.kind === "day") {
                return (
                  <Text variant="caption" tone="subtle" center style={styles.day}>
                    {item.label}
                  </Text>
                );
              }
              const m = item.message;
              const mine = m.senderUserId === user?.id;
              return (
                <View style={[styles.bubbleRow, mine ? styles.right : styles.left]}>
                  <View
                    style={[
                      styles.bubble,
                      mine
                        ? { backgroundColor: colors.primary, borderBottomRightRadius: 6 }
                        : { backgroundColor: colors.surface, borderColor: colors.border, borderWidth: 1, borderBottomLeftRadius: 6 },
                      item.pending ? { opacity: 0.6 } : null
                    ]}
                  >
                    <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{m.content}</Text>
                  </View>
                  <Text variant="caption" tone="subtle" style={styles.time}>
                    {item.pending ? "Envoi…" : messageTime(m.createdAt)}
                    {mine && m.id === lastSeenOwnId ? " · Vu" : ""}
                  </Text>
                </View>
              );
            }}
          />
        )}
        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>
          <TextInput
            accessibilityLabel="Votre message"
            value={draft}
            onChangeText={setDraft}
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textSubtle}
            multiline
            maxLength={2000}
            style={[styles.input, typeScale.body, { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border }]}
            testID="chat-input"
          />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Envoyer"
            disabled={!draft.trim()}
            onPress={() => void send()}
            style={[styles.send, { backgroundColor: draft.trim() ? colors.primary : colors.surfaceMuted }]}
            testID="chat-send"
          >
            <Icon name="arrow-up" size={22} color={draft.trim() ? colors.onPrimary : colors.textSubtle} />
          </Pressable>
        </View>
      </KeyboardAvoidingView>
      <SafetySheet
        visible={sheet}
        onClose={() => setSheet(false)}
        userId={userId}
        firstName={firstName}
        matchId={match?.id}
        onDone={() => navigation.navigate("Tabs")}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.xs,
    paddingVertical: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  headerUser: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: { padding: spacing.md, gap: spacing.xs },
  day: { marginVertical: spacing.sm, textTransform: "capitalize" },
  bubbleRow: { maxWidth: "80%", gap: 2 },
  left: { alignSelf: "flex-start" },
  right: { alignSelf: "flex-end", alignItems: "flex-end" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.xs + 2 },
  time: { paddingHorizontal: 4 },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: spacing.xs, padding: spacing.sm, borderTopWidth: StyleSheet.hairlineWidth },
  input: { flex: 1, maxHeight: 120, minHeight: 44, borderWidth: 1, borderRadius: 22, paddingHorizontal: spacing.md, paddingTop: 10, paddingBottom: 10 },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
