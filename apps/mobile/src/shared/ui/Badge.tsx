import React from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { colors, radii } from "./tokens";

export function CountBadge({ count }: { count: number }) {
  if (count <= 0) {
    return null;
  }
  return (
    <View accessibilityLabel={`${count} non lus`} style={styles.badge}>
      <Text style={styles.label} tone="onPrimary" variant="caption" weight="bold">
        {count > 99 ? "99+" : String(count)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    minWidth: 22,
    height: 22,
    paddingHorizontal: 6,
    borderRadius: radii.pill,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center"
  },
  label: {
    lineHeight: 15
  }
});
