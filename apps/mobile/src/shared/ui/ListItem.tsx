import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { spacing } from "./tokens";

type ListItemProps = {
  title: string;
  subtitle?: string;
  subtitleBold?: boolean;
  meta?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
  onPress?: () => void;
  onLongPress?: () => void;
  testID?: string;
};

export function ListItem({ title, subtitle, subtitleBold, meta, leading, trailing, onPress, onLongPress, testID }: ListItemProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={[title, subtitle].filter(Boolean).join(", ")}
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress && !onLongPress}
      onLongPress={onLongPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
      testID={testID}
    >
      {leading}
      <View style={styles.copy}>
        <View style={styles.titleRow}>
          <Text numberOfLines={1} style={styles.title} weight="semibold">
            {title}
          </Text>
          {meta ? (
            <Text tone="subtle" variant="caption">
              {meta}
            </Text>
          ) : null}
        </View>
        {subtitle ? (
          <Text numberOfLines={1} tone={subtitleBold ? "default" : "muted"} variant="label" weight={subtitleBold ? "semibold" : "regular"}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xs,
    borderRadius: 16
  },
  copy: { flex: 1, gap: 2 },
  titleRow: { flexDirection: "row", alignItems: "baseline", justifyContent: "space-between", gap: spacing.sm },
  title: { flexShrink: 1 }
});
