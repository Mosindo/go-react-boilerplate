import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  AppState,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useQueryClient } from "@tanstack/react-query";
import { markConversationRead, sendMessage } from "../api/chat";
import { unmatch } from "../api/matching";
import type { ConversationSummary } from "../api/models";
import { blockUser } from "../api/safety";
import { BottomSheet } from "../components/BottomSheet";
import { errorMessage, errorStatus } from "../components/errors";
import { AppButton, Banner, IconButton, MIN_TARGET, Skeleton, Txt } from "../components/kit";
import { MessageBubble } from "../components/MessageBubble";
import { PersonAvatar } from "../components/PersonAvatar";
import { ProfileModal } from "../components/ProfileModal";
import { ReportSheet } from "../components/ReportSheet";
import {
  applyMessageToConversations,
  removeConversation,
  setConversationUnread
} from "../lib/dating/cache";
import {
  MAX_MESSAGE_LENGTH,
  buildThreadItems,
  canSend,
  classifySendFailure,
  flattenMessages,
  showCounter,
  upsertMessage,
  type PendingMessage,
  type ThreadItem
} from "../lib/dating/messages";
import {
  CONVERSATIONS_KEY,
  messagesKey,
  useMessagesQuery,
  type ConversationsData,
  type MessagesData
} from "../realtime/queries";
import { useRealtime, useRealtimeEvents } from "../realtime/RealtimeProvider";
import { useAuth } from "../hooks/useAuth";
import { useTheme } from "../shared/ui/theme";

type Props = {
  conversation: ConversationSummary;
  onBack: () => void;
  /** The conversation cannot be used anymore (unmatched, blocked, deleted). */
  onGone: (message?: string) => void;
};

const READ_DEBOUNCE_MS = 400;

let localCounter = 0;
function newLocalId(): string {
  localCounter += 1;
  return `local-${Date.now()}-${localCounter}`;
}

export default function ChatThreadScreen({ conversation, onBack, onGone }: Props) {
  const { colors, spacing, radii } = useTheme();
  const { user } = useAuth();
  const myUserId = user?.id ?? "";
  const queryClient = useQueryClient();
  const { setActiveConversation } = useRealtime();
  const conversationId = conversation.id;
  const other = conversation.user;

  const messagesQuery = useMessagesQuery(conversationId);
  const pages = messagesQuery.data?.pages;
  const serverMessages = useMemo(() => (pages ? flattenMessages(pages) : []), [pages]);

  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [banner, setBanner] = useState<string | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  const listRef = useRef<FlatList<ThreadItem>>(null);
  const readTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const sending = useRef(new Set<string>());

  const items = useMemo(
    () => buildThreadItems(serverMessages, pending, myUserId),
    [serverMessages, pending, myUserId]
  );

  // ---- open thread bookkeeping -------------------------------------------------------------
  useEffect(() => {
    setActiveConversation(conversationId);
    return () => setActiveConversation(null);
  }, [conversationId, setActiveConversation]);

  const markReadNow = useCallback(async () => {
    try {
      await markConversationRead(conversationId);
      queryClient.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
        d ? setConversationUnread(d, conversationId, 0) : d
      );
    } catch {
      // best effort: the next event or foreground retries
    }
  }, [conversationId, queryClient]);

  const markReadSoon = useCallback(() => {
    if (readTimer.current) clearTimeout(readTimer.current);
    readTimer.current = setTimeout(() => {
      readTimer.current = null;
      void markReadNow();
    }, READ_DEBOUNCE_MS);
  }, [markReadNow]);

  useEffect(() => {
    void markReadNow();
    return () => {
      if (readTimer.current) {
        clearTimeout(readTimer.current);
        readTimer.current = null;
        void markReadNow();
      }
    };
  }, [markReadNow]);

  useEffect(() => {
    const sub = AppState.addEventListener("change", (next) => {
      if (next !== "active") return;
      markReadSoon();
      void queryClient.invalidateQueries({
        queryKey: messagesKey(conversationId)
      });
    });
    return () => sub.remove();
  }, [conversationId, markReadSoon, queryClient]);

  useRealtimeEvents((event) => {
    if (event.type === "message.new") {
      const m = event.data.message;
      if (m.conversationId === conversationId && m.senderId !== myUserId) markReadSoon();
    } else if (event.type === "match.removed" && event.data.conversationId === conversationId) {
      onGone("This match is no longer available.");
    }
  });

  // 404 while loading messages: the conversation is gone.
  const loadStatus = messagesQuery.isError ? errorStatus(messagesQuery.error) : null;
  useEffect(() => {
    if (loadStatus === 404) onGone("This conversation is no longer available.");
  }, [loadStatus, onGone]);

  // ---- sending ------------------------------------------------------------------------------
  const deliver = useCallback(
    async (localId: string, body: string) => {
      if (sending.current.has(localId)) return;
      sending.current.add(localId);
      setPending((list) =>
        list.map((p) => (p.localId === localId ? { ...p, status: "sending", error: undefined } : p))
      );
      try {
        const message = await sendMessage(conversationId, body);
        if (queryClient.getQueryData<MessagesData>(messagesKey(conversationId))) {
          queryClient.setQueryData<MessagesData>(messagesKey(conversationId), (d) =>
            d ? upsertMessage(d, message) : d
          );
        } else {
          void queryClient.invalidateQueries({
            queryKey: messagesKey(conversationId)
          });
        }
        queryClient.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
          d
            ? applyMessageToConversations(d, message, {
                myUserId,
                isOpen: true
              }).data
            : d
        );
        setPending((list) => list.filter((p) => p.localId !== localId));
      } catch (e) {
        const failure = classifySendFailure(errorStatus(e));
        if (failure === "gone") {
          onGone("This conversation is no longer available.");
          return;
        }
        const text =
          failure === "forbidden"
            ? "You can't message this user."
            : "Not sent. Check your connection and tap the message to retry.";
        if (failure === "forbidden") setBanner(text);
        setPending((list) =>
          list.map((p) => (p.localId === localId ? { ...p, status: "failed", error: text } : p))
        );
      } finally {
        sending.current.delete(localId);
      }
    },
    [conversationId, myUserId, onGone, queryClient]
  );

  const onSend = useCallback(() => {
    if (!canSend(draft)) return;
    const body = draft.trim();
    const localId = newLocalId();
    setPending((list) => [
      ...list,
      { localId, body, createdAt: new Date().toISOString(), status: "sending" }
    ]);
    setDraft("");
    setBanner(null);
    listRef.current?.scrollToOffset({ offset: 0, animated: true });
    void deliver(localId, body);
  }, [deliver, draft]);

  const onRetry = useCallback(
    (localId: string) => {
      const target = pending.find((p) => p.localId === localId);
      if (target) void deliver(localId, target.body);
    },
    [deliver, pending]
  );

  const onDiscard = useCallback((localId: string) => {
    Alert.alert("Discard message?", "This message was not sent.", [
      { text: "Keep", style: "cancel" },
      {
        text: "Discard",
        style: "destructive",
        onPress: () => setPending((list) => list.filter((p) => p.localId !== localId))
      }
    ]);
  }, []);

  // ---- menu actions -------------------------------------------------------------------------
  const cleanupAndLeave = useCallback(
    (message: string) => {
      queryClient.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
        d ? removeConversation(d, conversationId) : d
      );
      queryClient.removeQueries({ queryKey: messagesKey(conversationId) });
      void queryClient.invalidateQueries({ queryKey: CONVERSATIONS_KEY });
      onGone(message);
    },
    [conversationId, onGone, queryClient]
  );

  const runDestructive = useCallback(
    async (action: () => Promise<void>, doneMessage: string, failMessage: string) => {
      setBusy(true);
      setBanner(null);
      try {
        await action();
        cleanupAndLeave(doneMessage);
      } catch (e) {
        if (errorStatus(e) === 404) cleanupAndLeave(doneMessage);
        else setBanner(errorMessage(e, failMessage));
      } finally {
        setBusy(false);
      }
    },
    [cleanupAndLeave]
  );

  const afterMenuClose = useCallback((fn: () => void) => {
    setMenuOpen(false);
    // iOS cannot present an alert or a second modal while the sheet is still dismissing.
    setTimeout(fn, Platform.OS === "ios" ? 400 : 0);
  }, []);

  const confirmUnmatch = useCallback(() => {
    afterMenuClose(() =>
      Alert.alert(
        `Unmatch with ${other.firstName}?`,
        "The conversation will be deleted for both of you. This can't be undone.",
        [
          { text: "Cancel", style: "cancel" },
          {
            text: "Unmatch",
            style: "destructive",
            onPress: () =>
              void runDestructive(
                () => unmatch(conversation.matchId),
                `You unmatched with ${other.firstName}.`,
                "We couldn't unmatch right now. Please try again."
              )
          }
        ]
      )
    );
  }, [afterMenuClose, conversation.matchId, other.firstName, runDestructive]);

  const confirmBlock = useCallback(() => {
    afterMenuClose(() =>
      Alert.alert(
        `Block ${other.firstName}?`,
        "You won't see each other anymore and the conversation will be removed.",
        [
          { text: "Cancel", style: "cancel" },
          {
            text: "Block",
            style: "destructive",
            onPress: () =>
              void runDestructive(
                () => blockUser(other.userId),
                `${other.firstName} was blocked.`,
                "We couldn't block this person right now. Please try again."
              )
          }
        ]
      )
    );
  }, [afterMenuClose, other.firstName, other.userId, runDestructive]);

  const openProfile = useCallback(
    () => afterMenuClose(() => setProfileOpen(true)),
    [afterMenuClose]
  );
  const openReport = useCallback(() => afterMenuClose(() => setReportOpen(true)), [afterMenuClose]);

  const onEndReached = useCallback(() => {
    if (messagesQuery.hasNextPage && !messagesQuery.isFetchingNextPage) {
      void messagesQuery.fetchNextPage();
    }
  }, [messagesQuery]);

  const renderItem = useCallback(
    ({ item }: { item: ThreadItem }) =>
      item.kind === "separator" ? (
        <View style={styles.separator} accessibilityRole="header">
          <Txt variant="caption" tone="subtle" weight="semibold">
            {item.label}
          </Txt>
        </View>
      ) : (
        <MessageBubble item={item} onRetry={onRetry} onDiscard={onDiscard} />
      ),
    [onRetry, onDiscard]
  );

  const draftLength = draft.length;
  const sendEnabled = canSend(draft);
  const initialLoading = messagesQuery.isPending;
  const loadFailed = messagesQuery.isError && serverMessages.length === 0 && loadStatus !== 404;
  const empty = !initialLoading && !loadFailed && items.length === 0;

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <View
        style={[
          styles.header,
          {
            borderBottomColor: colors.border,
            paddingHorizontal: spacing.sm,
            gap: spacing.xs
          }
        ]}
      >
        <IconButton
          glyph="‹"
          glyphSize={30}
          label="Back to matches"
          onPress={onBack}
          testID="chat-back"
        />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${other.firstName}. View profile`}
          onPress={openProfile}
          testID="chat-header-profile"
          style={[styles.headerProfile, { gap: spacing.sm, minHeight: MIN_TARGET }]}
        >
          <PersonAvatar name={other.firstName} photo={other.photo} size={40} />
          <Txt variant="subheading" weight="bold" numberOfLines={1} style={styles.flex}>
            {other.firstName}
          </Txt>
        </Pressable>
        <IconButton
          glyph="⋯"
          glyphSize={26}
          label="More options"
          onPress={() => setMenuOpen(true)}
          disabled={busy}
          testID="chat-menu"
        />
      </View>

      {banner ? (
        <Banner
          tone="danger"
          message={banner}
          onDismiss={() => setBanner(null)}
          testID="chat-banner"
          style={{ margin: spacing.sm }}
        />
      ) : null}

      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        style={styles.flex}
      >
        <View style={styles.flex}>
          {initialLoading ? (
            <View style={{ padding: spacing.lg, gap: spacing.md }} testID="chat-loading">
              <Skeleton style={{ height: 40, width: "60%", borderRadius: 18 }} />
              <Skeleton
                style={{
                  height: 40,
                  width: "45%",
                  borderRadius: 18,
                  alignSelf: "flex-end"
                }}
              />
              <Skeleton style={{ height: 56, width: "70%", borderRadius: 18 }} />
            </View>
          ) : loadFailed ? (
            <View style={styles.center} testID="chat-error">
              <Txt tone="muted" style={styles.centerText}>
                We couldn't load this conversation.
              </Txt>
              <AppButton
                label="Try again"
                variant="soft"
                onPress={() => void messagesQuery.refetch()}
              />
            </View>
          ) : empty ? (
            <View style={styles.center} testID="chat-empty">
              <PersonAvatar name={other.firstName} photo={other.photo} size={88} />
              <Txt variant="heading" style={styles.centerText}>
                {`You matched with ${other.firstName}`}
              </Txt>
              <Txt tone="muted" style={styles.centerText}>
                Say hello 👋 Break the ice with a friendly first message.
              </Txt>
            </View>
          ) : (
            <FlatList
              ref={listRef}
              testID="chat-messages"
              data={items}
              inverted
              keyExtractor={(i) => i.key}
              renderItem={renderItem}
              onEndReached={onEndReached}
              onEndReachedThreshold={0.4}
              keyboardShouldPersistTaps="handled"
              keyboardDismissMode="interactive"
              contentContainerStyle={{ paddingVertical: spacing.md }}
              ListFooterComponent={
                messagesQuery.isFetchingNextPage ? (
                  <ActivityIndicator color={colors.primary} style={{ padding: spacing.md }} />
                ) : null
              }
            />
          )}
        </View>

        <View
          style={[
            styles.composer,
            {
              borderTopColor: colors.border,
              padding: spacing.sm,
              gap: spacing.sm
            }
          ]}
        >
          <View style={styles.inputWrap}>
            <TextInput
              value={draft}
              onChangeText={setDraft}
              placeholder="Write a message"
              placeholderTextColor={colors.textSubtle}
              multiline
              maxLength={MAX_MESSAGE_LENGTH}
              accessibilityLabel="Message"
              testID="message-input"
              style={[
                styles.input,
                {
                  color: colors.text,
                  backgroundColor: colors.surface,
                  borderColor: colors.border,
                  borderRadius: radii.lg,
                  paddingHorizontal: spacing.md
                }
              ]}
            />
            {showCounter(draftLength) ? (
              <Txt
                variant="caption"
                tone={draftLength >= MAX_MESSAGE_LENGTH ? "danger" : "subtle"}
                style={styles.counter}
                accessibilityLiveRegion="polite"
              >
                {`${draftLength}/${MAX_MESSAGE_LENGTH}`}
              </Txt>
            ) : null}
          </View>
          <IconButton
            glyph="↑"
            glyphSize={22}
            label="Send message"
            onPress={onSend}
            disabled={!sendEnabled}
            size={46}
            color={colors.primaryForeground}
            backgroundColor={colors.primary}
            borderColor={colors.primary}
            testID="send-button"
          />
        </View>
      </KeyboardAvoidingView>

      <BottomSheet
        visible={menuOpen}
        onClose={() => setMenuOpen(false)}
        accessibilityLabel="Conversation options"
        testID="chat-menu-sheet"
      >
        <View style={{ padding: spacing.lg, gap: spacing.sm }}>
          <AppButton
            label="View profile"
            variant="outline"
            onPress={openProfile}
            testID="menu-view-profile"
            fullWidth
          />
          <AppButton
            label="Report"
            variant="outline"
            onPress={openReport}
            testID="menu-report"
            fullWidth
          />
          <AppButton
            label="Block"
            variant="danger"
            onPress={confirmBlock}
            testID="menu-block"
            fullWidth
          />
          <AppButton
            label="Unmatch"
            variant="danger"
            onPress={confirmUnmatch}
            testID="menu-unmatch"
            fullWidth
          />
          <AppButton label="Cancel" variant="ghost" onPress={() => setMenuOpen(false)} fullWidth />
        </View>
      </BottomSheet>

      <ProfileModal
        visible={profileOpen}
        userId={other.userId}
        onClose={() => setProfileOpen(false)}
        onBlocked={() => cleanupAndLeave(`${other.firstName} was blocked.`)}
      />

      <ReportSheet
        visible={reportOpen}
        userId={other.userId}
        firstName={other.firstName}
        onClose={() => setReportOpen(false)}
        onReported={({ blocked }) => {
          if (blocked) cleanupAndLeave(`${other.firstName} was blocked.`);
        }}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: 4,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  headerProfile: { flex: 1, flexDirection: "row", alignItems: "center" },
  separator: { alignItems: "center", paddingVertical: 10 },
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    padding: 32,
    gap: 12
  },
  centerText: { textAlign: "center" },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    borderTopWidth: StyleSheet.hairlineWidth
  },
  inputWrap: { flex: 1 },
  input: {
    minHeight: 46,
    maxHeight: 140,
    borderWidth: 1,
    paddingTop: 12,
    paddingBottom: 12,
    fontSize: 15,
    textAlignVertical: "top"
  },
  counter: { position: "absolute", right: 12, bottom: 2 }
});
