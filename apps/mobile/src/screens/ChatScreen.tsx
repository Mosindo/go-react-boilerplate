import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { StackActions, useFocusEffect } from "@react-navigation/native";
import { useInfiniteQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { fetchMessages, markConversationRead, sendMessage } from "../api/chat";
import { ErrorView } from "../shared/feedback";
import { Avatar, Notice, Text, radii, spacing } from "../shared/ui";
import {
  MAX_MESSAGE_LENGTH,
  buildChatItems,
  flattenPages,
  formatClock,
  mergeWithPending,
  validateDraft,
  type ChatListItem,
  type PendingMessage
} from "../features/chat/messages";
import { errorMessage, isApiErrorCode } from "../features/errors";
import { useConversationsQuery } from "../features/hooks";
import { openUserProfile, useAppNavigation, useChatParams } from "../features/nav";
import { queryKeys } from "../features/queryKeys";
import { ReportModal } from "../features/safety/ReportModal";
import { useSafetyActions } from "../features/safety/useSafetyActions";
import type { Conversation, Message, MessagesPage } from "../features/types";
import { ActionMenu, type MenuAction } from "../features/ui/ActionMenu";
import { photoUri } from "../features/ui/PhotoImage";
import { ScreenHeader } from "../features/ui/ScreenHeader";
import { Skeleton } from "../features/ui/Skeleton";
import { usePalette } from "../features/ui/palette";
import { useAuth } from "../hooks/useAuth";
import {
  applyMessageToConversations,
  applyMessageToPages,
  clearConversationUnread
} from "../realtime/events";
import { useRealtime, useRealtimeEvent } from "../realtime/RealtimeProvider";

type MessagesData = InfiniteData<MessagesPage, string | undefined>;

export default function ChatScreen() {
  const p = usePalette();
  const navigation = useAppNavigation();
  const queryClient = useQueryClient();
  const { conversationId, user: peer } = useChatParams();
  const { user: me } = useAuth();
  const myId = me?.id ?? "";
  const { setActiveConversation } = useRealtime();

  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [blocked, setBlocked] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const counterRef = useRef(0);

  useLayoutEffect(() => {
    navigation.setOptions({ headerShown: false });
  }, [navigation]);

  const leave = useCallback(() => navigation.dispatch(StackActions.popToTop()), [navigation]);
  const safety = useSafetyActions({ userId: peer.userId, name: peer.firstName, onDone: leave });

  const conversationsQuery = useConversationsQuery();
  const conversation = conversationsQuery.data?.find((c) => c.id === conversationId);
  const peerLastReadAt = conversation?.peerLastReadAt ?? null;

  const messagesQuery = useInfiniteQuery<
    MessagesPage,
    Error,
    MessagesData,
    ReturnType<typeof queryKeys.messages>,
    string | undefined
  >({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => fetchMessages(conversationId, pageParam),
    initialPageParam: undefined,
    getNextPageParam: (last) => last.nextCursor ?? undefined
  });

  const server = useMemo(() => flattenPages(messagesQuery.data?.pages ?? []), [messagesQuery.data]);
  const items = useMemo(
    () => buildChatItems(mergeWithPending(server, pending), myId, peerLastReadAt, new Date()),
    [server, pending, myId, peerLastReadAt]
  );

  const markRead = useCallback(() => {
    queryClient.setQueryData<Conversation[]>(queryKeys.conversations, (cur) =>
      cur ? clearConversationUnread(cur, conversationId) : cur
    );
    markConversationRead(conversationId).catch(() => undefined);
  }, [conversationId, queryClient]);

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      markRead();
      return () => setActiveConversation(null);
    }, [conversationId, markRead, setActiveConversation])
  );

  useRealtimeEvent("message", (message: Message) => {
    if (message.conversationId === conversationId && message.senderId !== myId) {
      markRead();
    }
  });

  useEffect(() => {
    if (messagesQuery.isSuccess && (conversation?.unreadCount ?? 0) > 0) {
      markRead();
    }
  }, [messagesQuery.isSuccess, conversation?.unreadCount, markRead]);

  const deliver = useCallback(
    async (item: PendingMessage) => {
      setPending((cur) => cur.map((m) => (m.localId === item.localId ? { ...m, status: "sending" } : m)));
      try {
        const saved = await sendMessage(conversationId, item.content);
        queryClient.setQueryData<MessagesData>(queryKeys.messages(conversationId), (cur) =>
          cur
            ? applyMessageToPages(cur, saved)
            : { pages: [{ items: [saved], nextCursor: null }], pageParams: [undefined] }
        );
        queryClient.setQueryData<Conversation[]>(queryKeys.conversations, (cur) =>
          cur ? applyMessageToConversations(cur, saved, myId, conversationId).conversations : cur
        );
        setPending((cur) => cur.filter((m) => m.localId !== item.localId));
      } catch (err) {
        if (isApiErrorCode(err, "blocked")) {
          setBlocked(true);
          setPending((cur) => cur.filter((m) => m.localId !== item.localId));
          return;
        }
        setPending((cur) => cur.map((m) => (m.localId === item.localId ? { ...m, status: "failed" } : m)));
      }
    },
    [conversationId, myId, queryClient]
  );

  const send = useCallback(() => {
    const { ok, content } = validateDraft(draft);
    if (!ok || blocked) {
      return;
    }
    counterRef.current += 1;
    const item: PendingMessage = {
      localId: `local-${Date.now()}-${counterRef.current}`,
      conversationId,
      senderId: myId,
      content,
      createdAt: new Date().toISOString(),
      status: "sending"
    };
    setPending((cur) => [item, ...cur]);
    setDraft("");
    void deliver(item);
  }, [blocked, conversationId, deliver, draft, myId]);

  const onFailedPress = useCallback(
    (localId: string) => {
      const item = pending.find((m) => m.localId === localId);
      if (!item) {
        return;
      }
      Alert.alert("Message not sent", "Check your connection and try again.", [
        { text: "Cancel", style: "cancel" },
        {
          text: "Delete",
          style: "destructive",
          onPress: () => setPending((cur) => cur.filter((m) => m.localId !== localId))
        },
        { text: "Retry", onPress: () => void deliver(item) }
      ]);
    },
    [deliver, pending]
  );

  const viewProfile = useCallback(
    () => openUserProfile(navigation, { userId: peer.userId, conversationId }),
    [conversationId, navigation, peer.userId]
  );

  const menuActions: MenuAction[] = [
    { key: "view-profile", label: "View profile", onPress: viewProfile },
    { key: "unmatch", label: "Unmatch", destructive: true, onPress: safety.confirmUnmatch },
    { key: "report", label: "Report", destructive: true, onPress: safety.openReport },
    { key: "block", label: "Block", destructive: true, onPress: safety.confirmBlock }
  ];

  const renderItem = ({ item }: { item: ChatListItem }) => {
    if (item.kind === "separator") {
      return (
        <View style={styles.separator}>
          <Text tone="muted" variant="caption" weight="medium">
            {item.label}
          </Text>
        </View>
      );
    }
    const m = item.message;
    const mine = m.senderId === myId;
    const failed = m.status === "failed";
    return (
      <View style={[styles.messageWrap, mine ? styles.mineWrap : styles.theirsWrap]}>
        <Pressable
          accessibilityHint={failed ? "Double tap to retry or delete" : undefined}
          accessibilityLabel={`${mine ? "You" : peer.firstName}: ${m.content}${failed ? ". Not sent" : ""}`}
          disabled={!failed}
          onPress={() => m.localId && onFailedPress(m.localId)}
          style={[
            styles.bubble,
            mine
              ? { backgroundColor: p.primary }
              : { backgroundColor: p.surface, borderColor: p.border, borderWidth: StyleSheet.hairlineWidth },
            failed ? { opacity: 0.7 } : null,
            m.status === "sending" ? { opacity: 0.75 } : null
          ]}
          testID={mine ? "chat-message-mine" : "chat-message-theirs"}
        >
          <Text style={{ color: mine ? p.onPrimary : p.text }}>{m.content}</Text>
        </Pressable>
        {failed ? (
          <Text tone="danger" variant="caption" weight="medium">
            Not sent. Tap to retry
          </Text>
        ) : (
          <View style={styles.meta}>
            {item.showTime ? (
              <Text tone="muted" variant="caption">
                {m.status === "sending" ? "Sending..." : formatClock(m.createdAt)}
              </Text>
            ) : null}
            {item.showSeen ? (
              <Text tone="primary" variant="caption" weight="bold">
                Seen
              </Text>
            ) : null}
          </View>
        )}
      </View>
    );
  };

  const draftCheck = validateDraft(draft);
  const photo = photoUri(peer.photo);

  let content: React.ReactNode;
  if (messagesQuery.isLoading) {
    content = (
      <View style={styles.pad} testID="chat-loading">
        {[0, 1, 2, 3].map((i) => (
          <Skeleton
            key={i}
            style={{
              height: 38,
              width: i % 2 === 0 ? "55%" : "40%",
              alignSelf: i % 2 === 0 ? "flex-start" : "flex-end",
              marginBottom: spacing.md,
              borderRadius: radii.lg
            }}
          />
        ))}
      </View>
    );
  } else if (messagesQuery.isError && !messagesQuery.data) {
    content = (
      <View style={styles.pad}>
        <ErrorView
          message={errorMessage(messagesQuery.error, "Couldn't load messages.")}
          onAction={() => void messagesQuery.refetch()}
          testID="chat-error"
        />
      </View>
    );
  } else {
    content = (
      <FlatList
        ListEmptyComponent={
          <View style={[styles.empty, { transform: [{ scaleY: -1 }] }]}>
            <Text style={styles.centerText} tone="muted">
              You matched with {peer.firstName}. Say hello!
            </Text>
          </View>
        }
        ListFooterComponent={
          messagesQuery.isFetchingNextPage ? <Skeleton style={styles.olderLoading} /> : null
        }
        contentContainerStyle={styles.listContent}
        data={items}
        inverted
        keyExtractor={(i) => i.key}
        keyboardShouldPersistTaps="handled"
        onEndReached={() => {
          if (messagesQuery.hasNextPage && !messagesQuery.isFetchingNextPage) {
            void messagesQuery.fetchNextPage();
          }
        }}
        onEndReachedThreshold={0.3}
        renderItem={renderItem}
        testID="chat-list"
      />
    );
  }

  return (
    <SafeAreaView
      edges={["top", "bottom"]}
      style={[styles.flex, { backgroundColor: p.background }]}
      testID="chat-screen"
    >
      <ScreenHeader
        leading={<Avatar name={peer.firstName} size={36} uri={photo} />}
        onBack={() => navigation.goBack()}
        onTitlePress={viewProfile}
        right={
          <Pressable
            accessibilityLabel="Conversation options"
            accessibilityRole="button"
            hitSlop={10}
            onPress={() => setMenuOpen(true)}
            style={styles.menuButton}
            testID="chat-menu-button"
          >
            <Text variant="heading" weight="bold">
              {"⋯"}
            </Text>
          </Pressable>
        }
        subtitle={peer.age ? `${peer.age}` : undefined}
        title={peer.firstName}
      />
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : "height"} style={styles.flex}>
        <View style={styles.flex}>{content}</View>
        {blocked ? (
          <Notice
            description="One of you has blocked the other, so messages can't be sent."
            style={styles.notice}
            title="You can't message this person"
            tone="warning"
          />
        ) : (
          <View style={[styles.composer, { borderTopColor: p.border, backgroundColor: p.background }]}>
            <TextInput
              accessibilityLabel="Message"
              maxLength={MAX_MESSAGE_LENGTH}
              multiline
              onChangeText={setDraft}
              placeholder="Write a message"
              placeholderTextColor={p.textMuted}
              style={[styles.input, { backgroundColor: p.surface, borderColor: p.border, color: p.text }]}
              testID="chat-message-input"
              value={draft}
            />
            <Pressable
              accessibilityLabel="Send message"
              accessibilityRole="button"
              accessibilityState={{ disabled: !draftCheck.ok }}
              disabled={!draftCheck.ok}
              onPress={send}
              style={[styles.send, { backgroundColor: p.primary, opacity: draftCheck.ok ? 1 : 0.4 }]}
              testID="chat-send-button"
            >
              <Text style={{ color: p.onPrimary }} weight="bold">
                Send
              </Text>
            </Pressable>
          </View>
        )}
        {!blocked && draft.length >= MAX_MESSAGE_LENGTH - 200 ? (
          <Text
            style={styles.counter}
            tone={draft.length >= MAX_MESSAGE_LENGTH ? "danger" : "muted"}
            variant="caption"
          >
            {draft.length}/{MAX_MESSAGE_LENGTH}
          </Text>
        ) : null}
      </KeyboardAvoidingView>
      <ActionMenu actions={menuActions} onClose={() => setMenuOpen(false)} visible={menuOpen} />
      <ReportModal
        busy={safety.busy}
        name={peer.firstName}
        onClose={safety.closeReport}
        onSubmit={(r, d) => void safety.submitReport(r, d)}
        visible={safety.reportVisible}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  pad: { padding: spacing.lg },
  listContent: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  separator: { alignItems: "center", paddingVertical: spacing.md },
  messageWrap: { marginVertical: 2, maxWidth: "82%" },
  mineWrap: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirsWrap: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  meta: { flexDirection: "row", gap: spacing.sm, paddingHorizontal: spacing.xs },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderRadius: radii.lg,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingTop: spacing.sm,
    paddingBottom: spacing.sm,
    fontSize: 16
  },
  send: {
    height: 44,
    paddingHorizontal: spacing.lg,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center"
  },
  counter: { textAlign: "right", paddingHorizontal: spacing.lg, paddingBottom: spacing.xs },
  notice: { margin: spacing.md },
  menuButton: { width: 40, height: 40, alignItems: "center", justifyContent: "center" },
  empty: { padding: spacing.xl },
  centerText: { textAlign: "center" },
  olderLoading: { height: 24, marginVertical: spacing.md }
});
