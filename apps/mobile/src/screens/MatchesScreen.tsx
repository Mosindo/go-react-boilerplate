import React, { useCallback, useMemo, useState } from "react";
import { Alert, FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { hideConversation } from "../api/chat";
import { EmptyView, ErrorView } from "../shared/feedback";
import { Avatar, Text, radii, spacing } from "../shared/ui";
import { formatListTime } from "../features/chat/messages";
import { errorMessage } from "../features/errors";
import { newMatchesWithoutMessages, useConversationsQuery, useMatchesQuery } from "../features/hooks";
import { openChat, useAppNavigation } from "../features/nav";
import { queryKeys } from "../features/queryKeys";
import type { Conversation, Match } from "../features/types";
import { photoUri } from "../features/ui/PhotoImage";
import { ScreenHeader } from "../features/ui/ScreenHeader";
import { Skeleton } from "../features/ui/Skeleton";
import { usePalette } from "../features/ui/palette";
import { useAuth } from "../hooks/useAuth";
import { removeConversation } from "../realtime/events";

const PAGE_SIZE = 20;

export default function MatchesScreen() {
  const p = usePalette();
  const navigation = useAppNavigation();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const myId = user?.id ?? "";
  const conversationsQuery = useConversationsQuery();
  const matchesQuery = useMatchesQuery();
  const [visible, setVisible] = useState(PAGE_SIZE);
  const [refreshing, setRefreshing] = useState(false);

  const hide = useMutation({
    mutationFn: (conversationId: string) => hideConversation(conversationId),
    onSuccess: (_void, conversationId) => {
      queryClient.setQueryData<Conversation[]>(queryKeys.conversations, (cur) =>
        cur ? removeConversation(cur, conversationId) : cur
      );
    },
    onError: (err) => Alert.alert("Couldn't hide conversation", errorMessage(err, "Please try again."))
  });

  const conversations = conversationsQuery.data;
  const matches = matchesQuery.data;

  const newMatches = useMemo<Match[]>(
    () => (conversations && matches ? newMatchesWithoutMessages(matches, conversations) : []),
    [conversations, matches]
  );
  const chats = useMemo(() => (conversations ?? []).filter((c) => c.lastMessage !== null), [conversations]);
  const shown = chats.slice(0, visible);

  const refresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([conversationsQuery.refetch(), matchesQuery.refetch()]);
    } finally {
      setRefreshing(false);
    }
  }, [conversationsQuery, matchesQuery]);

  const confirmHide = useCallback(
    (c: Conversation) => {
      Alert.alert(
        `Hide conversation with ${c.user.firstName}?`,
        "It disappears for you only and comes back if they message you again.",
        [
          { text: "Cancel", style: "cancel" },
          { text: "Hide", style: "destructive", onPress: () => hide.mutate(c.id) }
        ]
      );
    },
    [hide]
  );

  const loading = conversationsQuery.isLoading || matchesQuery.isLoading;
  const failed = (conversationsQuery.isError || matchesQuery.isError) && !conversations && !matches;
  const now = new Date();

  const header = (
    <View>
      {newMatches.length > 0 ? (
        <View style={styles.newSection} testID="new-matches">
          <Text style={styles.sectionTitle} tone="muted" variant="eyebrow" weight="bold">
            New matches
          </Text>
          <ScrollView contentContainerStyle={styles.newRow} horizontal showsHorizontalScrollIndicator={false}>
            {newMatches.map((m) => (
              <Pressable
                accessibilityLabel={`New match: ${m.user.firstName}. Say hello`}
                accessibilityRole="button"
                key={m.id}
                onPress={() => openChat(navigation, m.conversationId, m.user)}
                style={styles.newItem}
                testID={`new-match-${m.user.userId}`}
              >
                <Avatar
                  name={m.user.firstName}
                  size={64}
                  style={{ borderColor: p.primary, borderWidth: 2 }}
                  uri={photoUri(m.user.photo)}
                />
                <Text numberOfLines={1} style={styles.newName} variant="caption" weight="medium">
                  {m.user.firstName}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>
      ) : null}
      {chats.length > 0 ? (
        <Text
          style={[styles.sectionTitle, styles.messagesTitle]}
          tone="muted"
          variant="eyebrow"
          weight="bold"
        >
          Messages
        </Text>
      ) : null}
    </View>
  );

  let body: React.ReactNode;
  if (loading) {
    body = (
      <View style={styles.skeletons} testID="matches-loading">
        <View style={styles.newRow}>
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} style={{ width: 64, height: 64, borderRadius: 32 }} />
          ))}
        </View>
        {[0, 1, 2, 3, 4].map((i) => (
          <View key={i} style={styles.skeletonRow}>
            <Skeleton style={{ width: 52, height: 52, borderRadius: 26 }} />
            <View style={styles.flex}>
              <Skeleton style={{ height: 14, width: "40%" }} />
              <Skeleton style={{ height: 12, width: "75%", marginTop: spacing.sm }} />
            </View>
          </View>
        ))}
      </View>
    );
  } else if (failed) {
    body = (
      <View style={styles.pad}>
        <ErrorView
          message={errorMessage(
            conversationsQuery.error ?? matchesQuery.error,
            "Couldn't load your matches."
          )}
          onAction={() => void refresh()}
          testID="matches-error"
        />
      </View>
    );
  } else {
    body = (
      <FlatList
        ListEmptyComponent={
          newMatches.length === 0 ? (
            <View style={styles.pad}>
              <EmptyView
                message="When you and someone else like each other, you'll find them here. Keep swiping in Discover."
                testID="matches-empty"
                title="No matches yet"
              />
            </View>
          ) : (
            <Text style={styles.pad} tone="muted">
              Say hello to one of your new matches.
            </Text>
          )
        }
        ListHeaderComponent={header}
        contentContainerStyle={styles.listContent}
        data={shown}
        keyExtractor={(c) => c.id}
        onEndReached={() => setVisible((v) => (v < chats.length ? v + PAGE_SIZE : v))}
        onEndReachedThreshold={0.4}
        refreshControl={<RefreshControl onRefresh={() => void refresh()} refreshing={refreshing} />}
        renderItem={({ item }) => {
          const last = item.lastMessage;
          const unread = item.unreadCount > 0;
          const mine = last?.senderId === myId;
          return (
            <Pressable
              accessibilityActions={[{ name: "hide", label: "Hide conversation" }]}
              accessibilityHint="Long press to hide this conversation"
              accessibilityLabel={`${item.user.firstName}${unread ? `, ${item.unreadCount} unread` : ""}. ${last ? `${mine ? "You: " : ""}${last.content}` : ""}`}
              accessibilityRole="button"
              onAccessibilityAction={(e) => {
                if (e.nativeEvent.actionName === "hide") {
                  confirmHide(item);
                }
              }}
              onLongPress={() => confirmHide(item)}
              onPress={() => openChat(navigation, item.id, item.user)}
              style={styles.row}
              testID={`conversation-${item.user.userId}`}
            >
              <Avatar name={item.user.firstName} size={52} uri={photoUri(item.user.photo)} />
              <View style={styles.flex}>
                <View style={styles.rowTop}>
                  <Text numberOfLines={1} style={styles.flex} weight={unread ? "bold" : "semibold"}>
                    {item.user.firstName}
                  </Text>
                  <Text tone={unread ? "primary" : "muted"} variant="caption">
                    {formatListTime(last?.createdAt ?? item.updatedAt, now)}
                  </Text>
                </View>
                <View style={styles.rowTop}>
                  <Text
                    numberOfLines={1}
                    style={styles.flex}
                    tone={unread ? "default" : "muted"}
                    weight={unread ? "semibold" : "regular"}
                  >
                    {mine ? "You: " : ""}
                    {last?.content ?? ""}
                  </Text>
                  {unread ? (
                    <View
                      accessibilityLabel={`${item.unreadCount} unread`}
                      style={[styles.badge, { backgroundColor: p.primary }]}
                    >
                      <Text style={{ color: p.onPrimary }} variant="caption" weight="bold">
                        {item.unreadCount > 99 ? "99+" : item.unreadCount}
                      </Text>
                    </View>
                  ) : null}
                </View>
              </View>
              <Pressable
                accessibilityLabel={`Hide conversation with ${item.user.firstName}`}
                accessibilityRole="button"
                hitSlop={8}
                onPress={() => confirmHide(item)}
                style={styles.more}
              >
                <Text tone="muted" weight="bold">
                  {"⋯"}
                </Text>
              </Pressable>
            </Pressable>
          );
        }}
        testID="conversation-list"
      />
    );
  }

  return (
    <SafeAreaView
      edges={["top"]}
      style={[styles.flex, { backgroundColor: p.background }]}
      testID="matches-screen"
    >
      <ScreenHeader title="Matches" />
      {body}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  pad: { padding: spacing.lg },
  listContent: { paddingBottom: spacing.xxl },
  newSection: { paddingTop: spacing.md },
  sectionTitle: { paddingHorizontal: spacing.lg, marginBottom: spacing.sm },
  messagesTitle: { marginTop: spacing.lg },
  newRow: { flexDirection: "row", gap: spacing.md, paddingHorizontal: spacing.lg },
  newItem: { width: 72, alignItems: "center", gap: spacing.xs },
  newName: { maxWidth: 72 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm
  },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radii.pill,
    paddingHorizontal: 6,
    alignItems: "center",
    justifyContent: "center"
  },
  more: { paddingHorizontal: spacing.xs, paddingVertical: spacing.sm },
  skeletons: { paddingTop: spacing.lg, gap: spacing.lg },
  skeletonRow: { flexDirection: "row", gap: spacing.md, alignItems: "center", paddingHorizontal: spacing.lg }
});
