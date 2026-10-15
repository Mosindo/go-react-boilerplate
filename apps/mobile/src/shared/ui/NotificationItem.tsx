import React from "react";
import { Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { AppNotification } from "../../api/types";
import { formatListTime } from "../../lib/dates";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type NotificationItemProps = {
  notification: AppNotification;
  onPress: () => void;
};

export function NotificationItem({ notification, onPress }: NotificationItemProps) {
  const palette = useTheme();
  const isMatch = notification.type === "match";
  const unread = !notification.isRead;

  return (
    <Pressable
      accessibilityHint="Ouvre la conversation"
      accessibilityLabel={`${notification.title}. ${notification.body}${unread ? ". Non lu" : ""}`}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => ({
        flexDirection: "row",
        alignItems: "center",
        gap: spacing.md,
        padding: spacing.md,
        borderRadius: radii.md,
        backgroundColor: unread ? palette.primarySoft : palette.surface,
        borderWidth: 1,
        borderColor: unread ? "transparent" : palette.border,
        opacity: pressed ? 0.85 : 1
      })}
    >
      <View
        style={{
          width: 44,
          height: 44,
          borderRadius: 22,
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: isMatch ? palette.primary : palette.brandSoft
        }}
      >
        <Ionicons color={isMatch ? palette.onPrimary : palette.brand} name={isMatch ? "heart" : "chatbubble-ellipses"} size={22} />
      </View>
      <View style={{ flex: 1, gap: 2 }}>
        <Text numberOfLines={1} weight={unread ? "bold" : "semibold"}>
          {notification.title}
        </Text>
        <Text numberOfLines={2} tone="muted" variant="label">
          {notification.body}
        </Text>
      </View>
      <Text tone="subtle" variant="caption">
        {formatListTime(notification.createdAt)}
      </Text>
    </Pressable>
  );
}
