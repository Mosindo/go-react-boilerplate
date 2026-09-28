import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { AppNotification } from "../../api/types";
import { formatListTime } from "../../domain/chat";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export function NotificationItem({
  notification,
  onPress
}: {
  notification: AppNotification;
  onPress: (n: AppNotification) => void;
}) {
  const theme = useTheme();
  const unread = notification.readAt === null;
  return (
    <Pressable
      accessibilityLabel={`${unread ? "Unread. " : ""}${notification.title}. ${notification.body}`}
      accessibilityRole="button"
      onPress={() => onPress(notification)}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: pressed ? theme.surfaceAlt : unread ? theme.primarySoft : "transparent" }
      ]}
    >
      <View style={[styles.dot, { backgroundColor: unread ? theme.primary : "transparent" }]} />
      <View style={styles.body}>
        <Text variant="label">{notification.title}</Text>
        <Text numberOfLines={2} tone="muted">
          {notification.body}
        </Text>
      </View>
      <Text tone="muted" variant="caption">
        {formatListTime(notification.createdAt)}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    minHeight: 64,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderRadius: radius.md
  },
  dot: { width: 10, height: 10, borderRadius: 5 },
  body: { flex: 1, gap: 2 }
});
