import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import type { Conversation } from "../api/types";
import { timeLabel } from "../lib/format";
import { MIN_TOUCH, spacing, useTheme } from "../theme";
import { AuthImage } from "./AuthImage";
import { Badge } from "./Badge";
import { EmptyState } from "./Feedback";
import { Text } from "./Text";

type Props = {
  conversations: Conversation[];
  myUserId: string;
  onOpen: (conversation: Conversation) => void;
  refreshing?: boolean;
  onRefresh?: () => void;
  onEndReached?: () => void;
  header?: React.ReactElement | null;
  emptyAction?: { label: string; onPress: () => void };
};

export function ConversationList({ conversations, myUserId, onOpen, refreshing, onRefresh, onEndReached, header, emptyAction }: Props) {
  const { colors } = useTheme();
  return (
    <FlatList
      testID="conversation-list"
      data={conversations}
      keyExtractor={(c) => c.id}
      refreshing={refreshing}
      onRefresh={onRefresh}
      onEndReached={onEndReached}
      onEndReachedThreshold={0.5}
      ListHeaderComponent={header}
      ListEmptyComponent={
        <EmptyState
          testID="conversations-empty"
          title="No conversations yet"
          message="When you and someone like each other, your conversation starts here."
          actionLabel={emptyAction?.label}
          onAction={emptyAction?.onPress}
        />
      }
      contentContainerStyle={conversations.length === 0 ? styles.emptyGrow : undefined}
      renderItem={({ item }) => {
        const last = item.lastMessage;
        const preview = last ? `${last.senderId === myUserId ? "You: " : ""}${last.body}` : "Say hello 👋";
        return (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Conversation with ${item.user.firstName}${item.unreadCount ? `, ${item.unreadCount} unread` : ""}`}
            onPress={() => onOpen(item)}
            testID={`conversation-${item.id}`}
            style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent", borderBottomColor: colors.border }]}
          >
            <AuthImage url={item.user.photoUrl} style={styles.avatar} fallbackInitial={item.user.firstName} />
            <View style={styles.flex}>
              <View style={styles.top}>
                <Text variant="label" style={[styles.flex, item.unreadCount > 0 && { fontWeight: "800" }]} numberOfLines={1}>
                  {item.user.firstName}
                </Text>
                <Text variant="caption" tone="muted">
                  {timeLabel(last?.createdAt ?? item.matchedAt)}
                </Text>
              </View>
              <View style={styles.top}>
                <Text variant="caption" tone={item.unreadCount > 0 ? "default" : "muted"} numberOfLines={1} style={styles.flex}>
                  {preview}
                </Text>
                <Badge count={item.unreadCount} />
              </View>
            </View>
          </Pressable>
        );
      }}
    />
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  emptyGrow: { flexGrow: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md, minHeight: MIN_TOUCH + 24, borderBottomWidth: StyleSheet.hairlineWidth },
  avatar: { width: 56, height: 56, borderRadius: 28 },
  top: { flexDirection: "row", alignItems: "center", gap: spacing.sm }
});
