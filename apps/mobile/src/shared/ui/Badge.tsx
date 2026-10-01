import React from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii } from "./tokens";

/** Count bubble (unread messages / notifications). Renders nothing for 0. */
export function CountBadge({ count, max = 99 }: { count: number; max?: number }) {
  const { colors } = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View accessibilityLabel={`${count} non lus`} style={[styles.badge, { backgroundColor: colors.primary }]}>
      <Text variant="caption" weight="bold" style={{ color: colors.primaryForeground }}>
        {count > max ? `${max}+` : String(count)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    minWidth: 20,
    height: 20,
    borderRadius: radii.pill,
    paddingHorizontal: 6,
    alignItems: "center",
    justifyContent: "center"
  }
});
