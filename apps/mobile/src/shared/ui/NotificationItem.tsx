import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { formatListTime } from "../../lib/format";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type NotificationItemProps = {
  type: "match" | "message";
  title: string;
  body: string;
  createdAt: string;
  isRead: boolean;
  onPress?: () => void;
};

export function NotificationItem({ type, title, body, createdAt, isRead, onPress }: NotificationItemProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={`${title}. ${body}`}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.row, { backgroundColor: isRead ? "transparent" : colors.primarySoft, opacity: pressed ? 0.85 : 1 }]}
    >
      <View style={[styles.icon, { backgroundColor: colors.surface }]}>
        <Ionicons color={colors.primary} name={type === "match" ? "heart" : "chatbubble"} size={20} />
      </View>
      <View style={styles.copy}>
        <Text numberOfLines={1} weight={isRead ? "medium" : "bold"}>
          {title}
        </Text>
        <Text numberOfLines={2} tone="muted" variant="label">
          {body}
        </Text>
      </View>
      <Text tone="subtle" variant="caption">
        {formatListTime(createdAt)}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.md, borderRadius: radii.md },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  copy: { flex: 1, gap: 2 }
});
