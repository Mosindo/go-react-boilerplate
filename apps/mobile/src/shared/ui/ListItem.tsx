import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { MIN_TOUCH, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = {
  title: string;
  subtitle?: string;
  left?: React.ReactNode;
  right?: React.ReactNode;
  onPress?: () => void;
  onLongPress?: () => void;
  tone?: "default" | "danger";
  chevron?: boolean;
  testID?: string;
};

export function ListItem({ title, subtitle, left, right, onPress, onLongPress, tone = "default", chevron, testID }: Props) {
  const theme = useTheme();
  const content = (
    <View style={[styles.row, { borderBottomColor: theme.border }]}>
      {left}
      <View style={styles.body}>
        <Text variant="body" tone={tone === "danger" ? "danger" : "default"} numberOfLines={1} style={{ fontWeight: "600" }}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" tone="muted" numberOfLines={1}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
      {chevron ? <Ionicons name="chevron-forward" size={18} color={theme.textMuted} /> : null}
    </View>
  );
  if (!onPress && !onLongPress) return content;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={subtitle ? `${title}, ${subtitle}` : title}
      onPress={onPress}
      onLongPress={onLongPress}
      style={({ pressed }) => ({ opacity: pressed ? 0.7 : 1 })}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    minHeight: MIN_TOUCH + 12,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  body: { flex: 1, gap: 2 }
});
