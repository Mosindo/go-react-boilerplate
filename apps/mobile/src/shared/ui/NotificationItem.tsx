import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

export type NotificationItemProps = {
  title: string;
  body: string;
  time: string;
  unread: boolean;
  icon: "heart" | "chatbubble";
  onPress: () => void;
};

export function NotificationItem({ title, body, time, unread, icon, onPress }: NotificationItemProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [
        styles.root,
        {
          backgroundColor: unread ? colors.primarySoft : colors.surface,
          borderColor: unread ? colors.primaryBorder : colors.border,
          opacity: pressed ? 0.8 : 1
        }
      ]}
    >
      <View style={[styles.icon, { backgroundColor: unread ? colors.primary : colors.surfaceSubtle }]}>
        <Ionicons color={unread ? colors.primaryForeground : colors.textMuted} name={icon} size={18} />
      </View>
      <View style={styles.copy}>
        <Text weight="semibold">{title}</Text>
        <Text tone="muted" variant="label">
          {body}
        </Text>
        <Text tone="subtle" variant="caption">
          {time}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flexDirection: "row", gap: spacing.md, padding: spacing.md, borderRadius: radii.lg, borderWidth: 1 },
  icon: { width: 36, height: 36, borderRadius: 18, alignItems: "center", justifyContent: "center" },
  copy: { flex: 1, gap: 2 }
});
