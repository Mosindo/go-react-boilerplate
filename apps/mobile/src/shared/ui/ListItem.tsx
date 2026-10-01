import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { spacing } from "./tokens";

type ListItemProps = {
  title: string;
  subtitle?: string;
  left?: ReactNode;
  right?: ReactNode;
  onPress?: () => void;
  emphasized?: boolean;
  destructive?: boolean;
};

export function ListItem({ title, subtitle, left, right, onPress, emphasized, destructive }: ListItemProps) {
  const { colors } = useTheme();
  const body = (
    <View style={styles.row}>
      {left}
      <View style={styles.text}>
        <Text numberOfLines={1} tone={destructive ? "danger" : "default"} weight={emphasized ? "bold" : "semibold"}>
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone={emphasized ? "default" : "muted"} variant="label" weight={emphasized ? "semibold" : "regular"}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
    </View>
  );
  if (!onPress) {
    return body;
  }
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={({ pressed }) => [pressed && { backgroundColor: colors.surfaceMuted }]}>
      {body}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.md, paddingHorizontal: spacing.lg },
  text: { flex: 1, gap: 2 }
});
