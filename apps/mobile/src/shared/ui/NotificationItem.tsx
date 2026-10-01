import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

type NotificationItemProps = {
  kind: string;
  title: string;
  body: string;
  time: string;
  unread: boolean;
  onPress: () => void;
};

const icons: Record<string, React.ComponentProps<typeof Ionicons>["name"]> = {
  match: "heart",
  message: "chatbubble-ellipses"
};

export function NotificationItem({ kind, title, body, time, unread, onPress }: NotificationItemProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={`${unread ? "Non lue. " : ""}${title}. ${body}`}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: unread ? colors.surfaceAccent : colors.surface, borderColor: colors.border },
        pressed && { opacity: 0.85 }
      ]}
    >
      <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name={icons[kind] ?? "notifications"} size={20} />
      </View>
      <View style={styles.text}>
        <Text weight={unread ? "bold" : "semibold"}>{title}</Text>
        <Text tone="muted" variant="label">
          {body}
        </Text>
        <Text tone="subtle" variant="caption">
          {time}
        </Text>
      </View>
      {unread ? <View style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.md, borderRadius: radii.lg, borderWidth: 1 },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
