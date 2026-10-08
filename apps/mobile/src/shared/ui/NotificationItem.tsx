import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { formatRelative } from "../format";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type NotificationItemProps = { title: string; body: string; createdAt: string; isRead: boolean; onPress?: () => void };

export function NotificationItem({ title, body, createdAt, isRead, onPress }: NotificationItemProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={`${title}. ${body}`}
      accessibilityRole="button"
      onPress={onPress}
      style={[styles.row, { backgroundColor: isRead ? colors.surface : colors.primarySoft, borderColor: colors.border }]}
    >
      <View style={[styles.icon, { backgroundColor: colors.primary }]}>
        <Ionicons color={colors.onPrimary} name="heart" size={18} />
      </View>
      <View style={styles.body}>
        <Text weight="semibold">{title}</Text>
        <Text tone="muted" variant="bodySmall">
          {body}
        </Text>
        <Text tone="subtle" variant="caption">
          {formatRelative(createdAt)}
        </Text>
      </View>
      {!isRead ? <View accessibilityLabel="Non lu" style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", gap: spacing.md, padding: spacing.md, borderRadius: radii.lg, borderWidth: 1, alignItems: "center" },
  icon: { width: 36, height: 36, borderRadius: 18, alignItems: "center", justifyContent: "center" },
  body: { flex: 1, gap: 2 },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
