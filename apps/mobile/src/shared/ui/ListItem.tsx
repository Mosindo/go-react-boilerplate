import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { hitSlop, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type ListItemProps = {
  title: string;
  subtitle?: string;
  left?: ReactNode;
  right?: ReactNode;
  onPress?: () => void;
  tone?: "default" | "danger";
  accessibilityHint?: string;
  bold?: boolean;
};

/** Generic row used by lists and settings menus. */
export function ListItem({
  title,
  subtitle,
  left,
  right,
  onPress,
  tone = "default",
  accessibilityHint,
  bold = false
}: ListItemProps) {
  const theme = useTheme();
  const content = (
    <View style={styles.row}>
      {left}
      <View style={styles.body}>
        <Text
          numberOfLines={1}
          style={bold ? { fontWeight: "700" } : undefined}
          tone={tone === "danger" ? "danger" : "default"}
          variant="body"
        >
          {title}
        </Text>
        {subtitle ? (
          <Text
            numberOfLines={1}
            style={bold ? { fontWeight: "600", color: theme.text } : undefined}
            tone="muted"
            variant="caption"
          >
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
    </View>
  );
  if (!onPress) {
    return content;
  }
  return (
    <Pressable
      accessibilityHint={accessibilityHint}
      accessibilityLabel={subtitle ? `${title}. ${subtitle}` : title}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [{ backgroundColor: pressed ? theme.surfaceAlt : "transparent" }]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    minHeight: hitSlop + 16,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.lg
  },
  body: { flex: 1, gap: 2 }
});
