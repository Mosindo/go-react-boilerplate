import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { ConversationSummary } from "../api/types";
import { Text, colors, radii, spacing } from "../shared/ui";
import { formatMessageTime } from "../utils/format";
import { PhotoImage } from "./PhotoImage";

type ConversationRowProps = {
  conversation: ConversationSummary;
  currentUserId: string;
  onPress: () => void;
  onLongPress?: () => void;
};

export function ConversationRow({ conversation, currentUserId, onPress, onLongPress }: ConversationRowProps) {
  const { user, lastMessage, unreadCount } = conversation;
  const unread = unreadCount > 0;
  const preview = lastMessage
    ? `${lastMessage.senderId === currentUserId ? "Vous : " : ""}${lastMessage.content}`
    : "Dites bonjour 👋";

  return (
    <Pressable
      accessibilityLabel={`Conversation avec ${user.firstName}${unread ? `, ${unreadCount} non lu(s)` : ""}`}
      accessibilityRole="button"
      onLongPress={onLongPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
      testID={`conversation-${user.firstName}`}
    >
      <PhotoImage fallbackLabel={user.firstName} path={user.thumbUrl} style={styles.avatar} />
      <View style={styles.copy}>
        <View style={styles.top}>
          <Text numberOfLines={1} style={styles.name} weight={unread ? "bold" : "semibold"}>
            {user.firstName}
          </Text>
          <Text tone="muted" variant="caption">
            {formatMessageTime(lastMessage?.createdAt ?? conversation.createdAt)}
          </Text>
        </View>
        <Text numberOfLines={1} tone={unread ? "default" : "muted"} weight={unread ? "semibold" : "regular"}>
          {preview}
        </Text>
      </View>
      {unread ? (
        <View accessibilityLabel={`${unreadCount} non lus`} style={styles.badge}>
          <Text tone="inverse" variant="caption" weight="bold">
            {unreadCount > 99 ? "99+" : unreadCount}
          </Text>
        </View>
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg
  },
  pressed: { backgroundColor: colors.surfaceMuted },
  avatar: { width: 56, height: 56, borderRadius: 28 },
  copy: { flex: 1, gap: 2 },
  top: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", gap: spacing.sm },
  name: { flex: 1 },
  badge: {
    minWidth: 24,
    height: 24,
    borderRadius: radii.pill,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
