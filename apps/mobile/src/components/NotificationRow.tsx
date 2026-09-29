import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { AppNotification } from "../api/models";
import { formatRelativeTime } from "../lib/dating/format";
import { useTheme } from "../shared/ui/theme";
import { Txt } from "./kit";

type Props = {
  notification: AppNotification;
  onPress: (notification: AppNotification) => void;
};

const GLYPH: Record<AppNotification["type"], string> = {
  match: "♥",
  message: "✉"
};

function NotificationRowBase({ notification, onPress }: Props) {
  const { colors, radii, spacing } = useTheme();
  const unread = !notification.isRead;
  const time = formatRelativeTime(notification.createdAt);
  return (
    <Pressable
      testID={`notification-row-${notification.id}`}
      accessibilityRole="button"
      accessibilityLabel={`${notification.title}. ${notification.body}. ${time}${unread ? ". Unread" : ""}`}
      accessibilityState={{ selected: unread }}
      onPress={() => onPress(notification)}
      style={({ pressed }) => [
        styles.row,
        {
          minHeight: 64,
          padding: spacing.md,
          gap: spacing.md,
          borderRadius: radii.lg,
          backgroundColor: unread
            ? colors.primarySoft
            : pressed
              ? colors.surfaceMuted
              : "transparent"
        }
      ]}
    >
      <View
        style={[
          styles.icon,
          {
            backgroundColor: notification.type === "match" ? colors.primary : colors.secondary
          }
        ]}
        importantForAccessibility="no-hide-descendants"
      >
        <Txt weight="bold" style={{ color: colors.background }}>
          {GLYPH[notification.type]}
        </Txt>
      </View>
      <View style={styles.body}>
        <Txt weight={unread ? "bold" : "semibold"} numberOfLines={1}>
          {notification.title}
        </Txt>
        <Txt variant="label" tone="muted" numberOfLines={2}>
          {notification.body}
        </Txt>
      </View>
      <View style={styles.side}>
        <Txt
          variant="caption"
          tone={unread ? "primary" : "subtle"}
          weight={unread ? "bold" : "regular"}
        >
          {time}
        </Txt>
        {unread ? <View style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
      </View>
    </Pressable>
  );
}

export const NotificationRow = React.memo(NotificationRowBase);

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center" },
  icon: {
    width: 44,
    height: 44,
    borderRadius: 22,
    alignItems: "center",
    justifyContent: "center"
  },
  body: { flex: 1, gap: 2 },
  side: { alignItems: "flex-end", gap: 6 },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
