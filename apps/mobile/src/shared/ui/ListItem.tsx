import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import type { IconName } from "./Button";
import { Text } from "./Text";
import { spacing } from "./tokens";

type ListItemProps = {
  title: string;
  subtitle?: string;
  icon?: IconName;
  left?: ReactNode;
  right?: ReactNode;
  onPress?: () => void;
  destructive?: boolean;
  chevron?: boolean;
};

/** Settings-style row. Pressable rows get a chevron unless a custom `right` is given. */
export function ListItem({ title, subtitle, icon, left, right, onPress, destructive = false, chevron = true }: ListItemProps) {
  const { colors } = useTheme();
  const tone = destructive ? colors.danger : colors.text;
  const content = (
    <View style={[styles.row, { borderBottomColor: colors.border }]}>
      {left ?? (icon ? <Ionicons color={destructive ? colors.danger : colors.textMuted} name={icon} size={22} /> : null)}
      <View style={styles.body}>
        <Text style={{ color: tone }} weight="medium">
          {title}
        </Text>
        {subtitle ? (
          <Text tone="muted" variant="bodySmall">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ?? (onPress && chevron ? <Ionicons color={colors.textSubtle} name="chevron-forward" size={18} /> : null)}
    </View>
  );
  if (!onPress) return content;
  return (
    <Pressable accessibilityLabel={title} accessibilityRole="button" onPress={onPress} style={({ pressed }) => (pressed ? { opacity: 0.7 } : null)}>
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.md, borderBottomWidth: StyleSheet.hairlineWidth, minHeight: 56 },
  body: { flex: 1, gap: 2 }
});
