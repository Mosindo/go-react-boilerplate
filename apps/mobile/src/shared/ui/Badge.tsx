import React from "react";
import { StyleSheet, View } from "react-native";
import { radius, useTheme } from "../../theme";
import { Text } from "./Text";

export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View
      accessibilityLabel={`${count} non lu${count > 1 ? "s" : ""}`}
      style={[styles.badge, { backgroundColor: colors.primary }]}
    >
      <Text style={{ color: colors.onPrimary }} variant="caption">
        {count > 99 ? "99+" : String(count)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radius.pill,
    paddingHorizontal: 6,
    alignItems: "center",
    justifyContent: "center"
  }
});
