import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { Alert, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useFocusEffect } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { keys } from "../../api/keys";
import { deleteConversation, markConversationRead, sendMessage, unmatch } from "../../api/social";
import type { Message } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { ActionSheet, ErrorState, LoadingState, Text, useToast } from "../../components";
import { useMessages } from "../../hooks/queries";
import { errorMessage, timeLabel } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import {
  applyIncomingMessage,
  clearUnread,
  removeConversation,
  upsertMessage,
  type ConversationsCache,
  type MessagesCache
} from "../../realtime/cache";
import { setActiveConversation } from "../../realtime/RealtimeProvider";
import { MIN_TOUCH, radii, spacing, useTheme } from "../../theme";
import { useSafety } from "../../hooks/useSafety";

type Outgoing = { localId: string; body: string; status: "sending" | "failed"; createdAt: string };
type Row =
  | { kind: "message"; message: Message; showRead: boolean }
  | { kind: "outgoing"; outgoing: Outgoing };

export function ChatScreen({ navigation, route }: NativeStackScreenProps<MainStackParamList, "Chat">) {
  const { conversationId, name, userId } = route.params;
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { confirmBlock } = useSafety();
  const me = user?.id ?? "";
  const query = useMessages(conversationId);
  const [outbox, setOutbox] = useState<Outgoing[]>([]);
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);
  const counter = useRef(0);

  const messages = query.data?.pages.flatMap((p) => p.messages) ?? [];
  const newestId = messages[0]?.id;
  const newestFromOther = messages[0] && messages[0].senderId !== me;

  const markRead = useCallback(() => {
    queryClient.setQueryData<ConversationsCache>(keys.conversations, (c) => clearUnread(c, conversationId));
    markConversationRead(conversationId).catch(() => undefined);
  }, [conversationId, queryClient]);

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      markRead();
      return () => setActiveConversation(null);
    }, [conversationId, markRead])
  );

  useEffect(() => {
    if (newestFromOther) markRead();
  }, [newestId, newestFromOther, markRead]);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => (
        <Pressable accessibilityRole="button" accessibilityLabel="Conversation options" onPress={() => setMenu(true)} testID="chat-menu" hitSlop={12} style={styles.menuButton}>
          <Text style={{ fontSize: 24 }}>⋯</Text>
        </Pressable>
      )
    });
  }, [navigation, name]);

  const deliver = useCallback(
    async (item: Outgoing) => {
      setOutbox((o) => o.map((x) => (x.localId === item.localId ? { ...x, status: "sending" } : x)));
      try {
        const sent = await sendMessage(conversationId, item.body);
        queryClient.setQueryData<MessagesCache>(keys.messages(conversationId), (c) => upsertMessage(c, sent));
        const current = queryClient.getQueryData<ConversationsCache>(keys.conversations);
        const result = applyIncomingMessage(current, sent, me, conversationId);
        if (result.found) queryClient.setQueryData(keys.conversations, result.cache);
        setOutbox((o) => o.filter((x) => x.localId !== item.localId));
      } catch (error) {
        setOutbox((o) => o.map((x) => (x.localId === item.localId ? { ...x, status: "failed" } : x)));
        toast.show(errorMessage(error), "error");
      }
    },
    [conversationId, me, queryClient, toast]
  );

  const send = (): void => {
    const body = draft.trim();
    if (!body) return;
    counter.current += 1;
    const item: Outgoing = { localId: `local-${counter.current}`, body, status: "sending", createdAt: new Date().toISOString() };
    setDraft("");
    setOutbox((o) => [item, ...o]);
    void deliver(item);
  };

  const lastReadMine = messages.find((m) => m.senderId === me && m.readAt)?.id;
  const rows: Row[] = [
    ...outbox.map((outgoing): Row => ({ kind: "outgoing", outgoing })),
    ...messages.map((message): Row => ({ kind: "message", message, showRead: message.id === lastReadMine }))
  ];

  const leave = (): void => {
    queryClient.setQueryData<ConversationsCache>(keys.conversations, (c) => removeConversation(c, conversationId));
    void queryClient.invalidateQueries({ queryKey: keys.conversations });
    navigation.popToTop();
  };

  const confirm = (title: string, message: string, label: string, action: () => Promise<void>): void => {
    Alert.alert(title, message, [
      { text: "Cancel", style: "cancel" },
      {
        text: label,
        style: "destructive",
        onPress: () => {
          action().then(leave).catch((e: unknown) => toast.show(errorMessage(e), "error"));
        }
      }
    ]);
  };

  if (query.isPending) return <SafeAreaView style={styles.flex}><LoadingState /></SafeAreaView>;
  if (query.isError) {
    return <SafeAreaView style={styles.flex}><ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} /></SafeAreaView>;
  }

  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: colors.background }]} testID="chat-screen">
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}>
        <FlatList
          inverted
          data={rows}
          keyExtractor={(row) => (row.kind === "message" ? row.message.id : row.outgoing.localId)}
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm }}
          onEndReachedThreshold={0.3}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          ListFooterComponent={query.isFetchingNextPage ? <LoadingState /> : null}
          ListEmptyComponent={
            <View style={styles.emptyWrap}>
              <Text tone="muted" style={{ textAlign: "center" }}>
                You matched with {name}. Break the ice!
              </Text>
            </View>
          }
          renderItem={({ item }) => {
            if (item.kind === "outgoing") {
              const failed = item.outgoing.status === "failed";
              return (
                <Pressable
                  disabled={!failed}
                  onPress={() => void deliver(item.outgoing)}
                  onLongPress={() => setOutbox((o) => o.filter((x) => x.localId !== item.outgoing.localId))}
                  accessibilityLabel={failed ? "Message failed. Tap to retry, long press to discard." : "Sending"}
                  style={[styles.bubble, styles.mine, { backgroundColor: colors.bubbleMine, opacity: failed ? 1 : 0.6 }]}
                >
                  <Text style={{ color: colors.bubbleMineText }}>{item.outgoing.body}</Text>
                  <Text variant="caption" style={{ color: failed ? colors.danger : colors.bubbleMineText, alignSelf: "flex-end" }}>
                    {failed ? "Not sent · tap to retry" : "Sending…"}
                  </Text>
                </Pressable>
              );
            }
            const { message, showRead } = item;
            const mine = message.senderId === me;
            return (
              <View style={mine ? styles.mineWrap : styles.theirsWrap}>
                <View style={[styles.bubble, mine ? styles.mine : styles.theirs, { backgroundColor: mine ? colors.bubbleMine : colors.bubbleTheirs }]}>
                  <Text style={{ color: mine ? colors.bubbleMineText : colors.text }}>{message.body}</Text>
                  <Text variant="caption" style={{ color: mine ? colors.bubbleMineText : colors.textMuted, alignSelf: "flex-end", opacity: 0.8 }}>
                    {timeLabel(message.createdAt)}
                  </Text>
                </View>
                {mine && showRead ? (
                  <Text variant="caption" tone="muted" testID="read-receipt">
                    Read
                  </Text>
                ) : null}
              </View>
            );
          }}
        />
        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.surface }]}>
          <TextInput
            value={draft}
            onChangeText={setDraft}
            placeholder="Write a message…"
            placeholderTextColor={colors.textMuted}
            accessibilityLabel="Message"
            multiline
            maxLength={2000}
            testID="chat-input"
            style={[styles.input, { color: colors.text, backgroundColor: colors.surfaceMuted }]}
          />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Send message"
            disabled={!draft.trim()}
            onPress={send}
            testID="chat-send"
            style={[styles.send, { backgroundColor: colors.primary, opacity: draft.trim() ? 1 : 0.4 }]}
          >
            <Text style={{ color: colors.primaryText, fontWeight: "700" }}>➤</Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>
      <ActionSheet
        visible={menu}
        title={name}
        onClose={() => setMenu(false)}
        actions={[
          { label: "View profile", onPress: () => navigation.navigate("ProfileDetail", { userId }) },
          { label: "Delete conversation", onPress: () => confirm("Delete this conversation?", "It is removed for you only. It returns if they write again.", "Delete", () => deleteConversation(conversationId)) },
          { label: "Unmatch", destructive: true, onPress: () => confirm(`Unmatch with ${name}?`, "You will no longer see each other.", "Unmatch", () => unmatch(conversationId)) },
          { label: "Report", onPress: () => navigation.navigate("Report", { userId, name, conversationId }) },
          { label: "Block", destructive: true, onPress: () => confirmBlock(userId, name, () => navigation.popToTop()) }
        ]}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  menuButton: { minWidth: MIN_TOUCH, minHeight: MIN_TOUCH, alignItems: "center", justifyContent: "center" },
  emptyWrap: { padding: spacing.xl, transform: [{ scaleY: -1 }] },
  mineWrap: { alignItems: "flex-end", gap: 2 },
  theirsWrap: { alignItems: "flex-start" },
  bubble: { maxWidth: "80%", paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radii.md },
  mine: { alignSelf: "flex-end", borderBottomRightRadius: 4 },
  theirs: { borderBottomLeftRadius: 4 },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm, padding: spacing.sm, borderTopWidth: StyleSheet.hairlineWidth },
  input: { flex: 1, minHeight: MIN_TOUCH, maxHeight: 120, borderRadius: radii.lg, paddingHorizontal: spacing.lg, paddingTop: 12, paddingBottom: 12, fontSize: 16 },
  send: { width: MIN_TOUCH, height: MIN_TOUCH, borderRadius: MIN_TOUCH / 2, alignItems: "center", justifyContent: "center" }
});
