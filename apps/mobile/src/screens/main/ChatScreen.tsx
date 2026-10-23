import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  TextInput,
  View
} from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { markMatchRead, sendMessage } from "../../api";
import type { MatchesPage, Message, MessagesPage } from "../../api/types";
import {
  buildChatEntries,
  lastOwnMessageId,
  newLocalId,
  type ChatEntry,
  type PendingMessage
} from "../../features/outbox";
import { useModeration } from "../../features/useModeration";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { formatDayLabel, formatMessageTime, isDifferentDay } from "../../lib/dates";
import { validateMessage } from "../../lib/validation";
import { useCard, useMessages } from "../../queries";
import { activeChat } from "../../realtime/activeChat";
import {
  appendMessage,
  applyMessageNewToMatches,
  clearUnread,
  flattenMessages,
  type Paged
} from "../../realtime/cache";
import { queryKeys } from "../../session/queryKeys";
import { useSession } from "../../session/SessionProvider";
import { createStyles, useTheme } from "../../theme/ThemeProvider";
import {
  Button,
  EmptyState,
  ErrorState,
  Photo,
  ScalePressable,
  Screen,
  Skeleton,
  Text,
  useToast
} from "../../ui";
import type { MainStackParams } from "../navTypes";

type Props = NativeStackScreenProps<MainStackParams, "Chat">;

const useStyles = createStyles(
  ({ colors, spacing, radius, minTouch, fontFamily, contentMaxWidth }) => ({
    list: { paddingVertical: spacing.md, gap: 2 },
    day: { alignSelf: "center", paddingVertical: spacing.sm },
    bubbleRow: { flexDirection: "row", marginVertical: 2 },
    mine: { justifyContent: "flex-end" },
    bubble: {
      maxWidth: "80%",
      borderRadius: radius.md,
      paddingHorizontal: spacing.md,
      paddingVertical: spacing.sm
    },
    bubbleMine: { backgroundColor: colors.bubbleMine, borderBottomRightRadius: 4 },
    bubbleTheirs: { backgroundColor: colors.bubbleTheirs, borderBottomLeftRadius: 4 },
    meta: { flexDirection: "row", justifyContent: "flex-end", gap: spacing.xs, marginTop: 2 },
    composer: {
      flexDirection: "row",
      alignItems: "flex-end",
      gap: spacing.sm,
      paddingVertical: spacing.sm,
      borderTopWidth: 1,
      borderTopColor: colors.border,
      width: "100%",
      maxWidth: contentMaxWidth,
      alignSelf: "center"
    },
    input: {
      flex: 1,
      minHeight: minTouch,
      maxHeight: 120,
      borderRadius: radius.lg,
      borderWidth: 1.5,
      borderColor: colors.border,
      backgroundColor: colors.surface,
      color: colors.text,
      paddingHorizontal: spacing.lg,
      paddingTop: 12,
      paddingBottom: 12,
      fontSize: 16,
      fontFamily
    },
    headerTitle: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
    headerAvatar: { width: 32, height: 32, borderRadius: 16 }
  })
);

export default function ChatScreen({ navigation, route }: Props) {
  const styles = useStyles();
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { userId: myUserId } = useSession();
  const { matchId, userId: otherUserId, thumbUrl } = route.params;

  const needsProfile = !route.params.firstName || !thumbUrl;
  const profile = useCard(otherUserId ?? "", undefined, needsProfile);
  const firstName =
    route.params.firstName ?? (otherUserId ? profile.data?.firstName : undefined) ?? "";
  const avatarPath = thumbUrl ?? profile.data?.photos[0]?.thumbUrl ?? null;

  const messages = useMessages(matchId);
  const moderation = useModeration(() => navigation.goBack());
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [draft, setDraft] = useState("");
  const listRef = useRef<FlatList<ChatEntry>>(null);

  const serverMessages = useMemo(
    () => flattenMessages(messages.data as Paged<MessagesPage> | undefined),
    [messages.data]
  );
  const entries = useMemo(
    () => buildChatEntries(serverMessages, pending),
    [serverMessages, pending]
  );
  const lastMineId = lastOwnMessageId(serverMessages, myUserId);
  const newestIncomingId =
    serverMessages.find((message) => message.senderId !== myUserId)?.id ?? null;

  // Register as the open conversation, leave if the match disappears.
  useEffect(() => {
    activeChat.set(matchId);
    const stop = activeChat.onRemoved((removed) => {
      if (removed === matchId) navigation.goBack();
    });
    return () => {
      activeChat.set(null);
      stop();
    };
  }, [matchId, navigation]);

  // Mark as read on open and whenever a new incoming message lands while the chat is visible.
  useEffect(() => {
    if (!messages.isSuccess) return;
    queryClient.setQueryData<Paged<MatchesPage>>(queryKeys.matches, (data) =>
      clearUnread(data, matchId)
    );
    void markMatchRead(matchId).catch(() => {
      // non-critical: next open retries
    });
  }, [matchId, messages.isSuccess, newestIncomingId, queryClient]);

  useEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <View style={styles.headerTitle}>
          <Photo name={firstName} path={avatarPath} style={styles.headerAvatar} />
          <Text numberOfLines={1} variant="heading">
            {firstName || t.chat.unknownUser}
          </Text>
        </View>
      ),
      headerRight: () =>
        otherUserId ? (
          <Button
            accessibilityLabel={t.common.more}
            compact
            label="⋯"
            onPress={() =>
              moderation.openMenu(
                { matchId, userId: otherUserId, firstName: firstName || t.chat.unknownUser },
                {
                  withHide: true,
                  onViewProfile: () => navigation.navigate("ProfileDetail", { userId: otherUserId })
                }
              )
            }
            testID="chat-menu-button"
            variant="ghost"
          />
        ) : null
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [avatarPath, firstName, matchId, navigation, otherUserId, colors]);

  const deliver = useCallback(
    async (item: PendingMessage) => {
      try {
        const sent: Message = await sendMessage(matchId, item.content);
        queryClient.setQueryData<Paged<MessagesPage>>(queryKeys.messages(matchId), (data) =>
          appendMessage(data, sent)
        );
        const matches = queryClient.getQueryData<Paged<MatchesPage>>(queryKeys.matches);
        const next = applyMessageNewToMatches(matches, sent, myUserId, matchId);
        if (next) queryClient.setQueryData(queryKeys.matches, next);
        setPending((list) => list.filter((entry) => entry.localId !== item.localId));
      } catch (error) {
        setPending((list) =>
          list.map((entry) =>
            entry.localId === item.localId ? { ...entry, status: "failed" } : entry
          )
        );
        toast.error(errorMessage(error));
      }
    },
    [matchId, myUserId, queryClient, toast]
  );

  const send = () => {
    if (validateMessage(draft)) return;
    const item: PendingMessage = {
      localId: newLocalId(),
      content: draft.trim(),
      createdAt: new Date().toISOString(),
      status: "sending"
    };
    setDraft("");
    setPending((list) => [...list, item]);
    void deliver(item);
  };

  const retry = (item: PendingMessage) => {
    const resending = { ...item, status: "sending" as const };
    setPending((list) => list.map((entry) => (entry.localId === item.localId ? resending : entry)));
    void deliver(resending);
  };

  const renderItem = ({ item, index }: { item: ChatEntry; index: number }) => {
    const mine = item.kind === "pending" || item.message.senderId === myUserId;
    const content = item.kind === "pending" ? item.pending.content : item.message.content;
    const createdAt = item.kind === "pending" ? item.pending.createdAt : item.message.createdAt;
    const older = entries[index + 1];
    const olderAt = older
      ? older.kind === "pending"
        ? older.pending.createdAt
        : older.message.createdAt
      : null;
    const showDay = item.kind === "message" && (!olderAt || isDifferentDay(createdAt, olderAt));

    let receipt: string | null = null;
    let failed = false;
    if (item.kind === "pending") {
      failed = item.pending.status === "failed";
      receipt = failed ? t.chat.failed : t.chat.sending;
    } else if (mine && item.message.id === lastMineId) {
      receipt = item.message.readAt ? t.chat.read : t.chat.sent;
    }

    const bubble = (
      <View style={[styles.bubbleRow, mine ? styles.mine : null]}>
        <View
          style={[
            styles.bubble,
            mine ? styles.bubbleMine : styles.bubbleTheirs,
            failed ? { opacity: 0.6 } : null
          ]}
        >
          <Text style={{ color: mine ? colors.onBubbleMine : colors.onBubbleTheirs }}>
            {content}
          </Text>
          <View style={styles.meta}>
            <Text
              style={{ color: mine ? colors.onBubbleMine : colors.onBubbleTheirs, opacity: 0.75 }}
              variant="caption"
            >
              {formatMessageTime(createdAt)}
            </Text>
          </View>
        </View>
      </View>
    );

    return (
      <View>
        {showDay ? (
          <View style={styles.day}>
            <Text tone="muted" variant="caption">
              {formatDayLabel(createdAt)}
            </Text>
          </View>
        ) : null}
        {item.kind === "pending" && failed ? (
          <ScalePressable
            accessibilityHint={t.chat.failed}
            accessibilityRole="button"
            onPress={() => retry(item.pending)}
            testID="chat-retry-message"
          >
            {bubble}
          </ScalePressable>
        ) : (
          bubble
        )}
        {receipt ? (
          <Text
            accessibilityLiveRegion="polite"
            style={{ alignSelf: "flex-end", color: failed ? colors.danger : colors.textMuted }}
            testID={failed ? undefined : "chat-receipt"}
            variant="caption"
          >
            {receipt}
          </Text>
        ) : null}
      </View>
    );
  };

  let body: React.ReactNode;
  if (messages.isPending) {
    body = (
      <View style={{ gap: 12, paddingVertical: 16 }}>
        <Skeleton height={40} radius={16} width="60%" />
        <Skeleton height={40} radius={16} style={{ alignSelf: "flex-end" }} width="50%" />
        <Skeleton height={40} radius={16} width="70%" />
      </View>
    );
  } else if (messages.isError && entries.length === 0) {
    body = <ErrorState onRetry={() => messages.refetch()} title={t.chat.errorTitle} />;
  } else {
    body = (
      <FlatList
        contentContainerStyle={styles.list}
        data={entries}
        inverted
        keyExtractor={(entry) =>
          entry.kind === "pending" ? entry.pending.localId : entry.message.id
        }
        keyboardShouldPersistTaps="handled"
        ListEmptyComponent={
          <View style={{ transform: [{ scaleY: -1 }], flex: 1 }}>
            <EmptyState
              glyph="✉"
              message={t.chat.emptyText(firstName || t.chat.unknownUser)}
              title={t.chat.emptyTitle}
            />
          </View>
        }
        ListFooterComponent={
          messages.isFetchingNextPage ? (
            <ActivityIndicator color={colors.primary} style={{ padding: 16 }} />
          ) : null
        }
        onEndReached={() => {
          if (messages.hasNextPage && !messages.isFetchingNextPage) void messages.fetchNextPage();
        }}
        onEndReachedThreshold={0.3}
        ref={listRef}
        renderItem={renderItem}
        testID="chat-list"
      />
    );
  }

  const canSend = !validateMessage(draft);

  return (
    <Screen avoidKeyboard={false} edges={[]} testID="chat-screen">
      <KeyboardAvoidingView
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        style={{ flex: 1 }}
      >
        <View style={{ flex: 1 }}>{body}</View>
        <View style={styles.composer}>
          <TextInput
            accessibilityLabel={t.chat.placeholder}
            maxLength={2000}
            multiline
            onChangeText={setDraft}
            placeholder={t.chat.placeholder}
            placeholderTextColor={colors.textMuted}
            style={styles.input}
            testID="chat-message-input"
            value={draft}
          />
          <View style={{ width: 96 }}>
            <Button
              compact
              disabled={!canSend}
              label={t.chat.send}
              onPress={send}
              testID="chat-send-button"
            />
          </View>
        </View>
      </KeyboardAvoidingView>
    </Screen>
  );
}
