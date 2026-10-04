import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { MatchSummary } from "../../api/types";
import { formatRelative } from "../../lib/format";
import { MIN_TOUCH, spacing, useColors } from "../../theme";
import { Badge, Icon, ProfilePhoto, Text } from "../../ui";

type Props = {
  item: MatchSummary;
  myUserId: string;
  onPress: () => void;
  onMore: () => void;
};

export function previewOf(item: MatchSummary, myUserId: string): string {
  if (!item.lastMessage) return "Say hello";
  return item.lastMessage.senderId === myUserId ? `You: ${item.lastMessage.body}` : item.lastMessage.body;
}

export function ConversationRow({ item, myUserId, onPress, onMore }: Props) {
  const colors = useColors();
  const unread = item.unreadCount > 0;
  const preview = previewOf(item, myUserId);
  return (
    <View style={styles.row}>
      <Pressable
        accessibilityActions={[{ name: "more", label: "More options" }]}
        accessibilityLabel={`${item.user.displayName}. ${preview}.${unread ? ` ${item.unreadCount} unread.` : ""}`}
        accessibilityRole="button"
        onAccessibilityAction={(e) => e.nativeEvent.actionName === "more" && onMore()}
        onLongPress={onMore}
        onPress={onPress}
        style={({ pressed }) => [styles.main, pressed && { backgroundColor: colors.surfaceMuted }]}
      >
        <ProfilePhoto name={item.user.displayName} photo={item.user.photo} round size="thumb" style={styles.avatar} />
        <View style={styles.text}>
          <View style={styles.top}>
            <Text numberOfLines={1} style={styles.name} weight={unread ? "bold" : "semibold"}>
              {item.user.displayName}
            </Text>
            <Text tone="muted" variant="caption">
              {formatRelative(item.lastMessage?.createdAt ?? item.updatedAt)}
            </Text>
          </View>
          <View style={styles.top}>
            <Text
              numberOfLines={1}
              style={styles.name}
              tone={unread ? "default" : "muted"}
              variant="small"
              weight={unread ? "semibold" : "regular"}
            >
              {preview}
            </Text>
            <Badge count={item.unreadCount} />
          </View>
        </View>
      </Pressable>
      <Pressable
        accessibilityLabel={`More options for ${item.user.displayName}`}
        accessibilityRole="button"
        hitSlop={4}
        onPress={onMore}
        style={styles.more}
      >
        <Icon color={colors.textMuted} name="ellipsis-horizontal" size={20} />
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center" },
  main: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingVertical: spacing.sm,
    paddingLeft: spacing.md,
    minHeight: 72
  },
  avatar: { width: 56, height: 56 },
  text: { flex: 1, gap: 2 },
  top: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.xs },
  name: { flexShrink: 1 },
  more: { width: MIN_TOUCH, height: MIN_TOUCH, alignItems: "center", justifyContent: "center" }
});
