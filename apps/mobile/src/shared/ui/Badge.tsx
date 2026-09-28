import React from "react";
import { StyleSheet, View } from "react-native";
import { radius, useTheme } from "../../theme";
import { Text } from "./Text";

/** Compact numeric badge, hidden when count is 0. */
export function Badge({ count, label }: { count: number; label?: string }) {
  const theme = useTheme();
  if (count <= 0) {
    return null;
  }
  const text = count > 99 ? "99+" : String(count);
  return (
    <View
      accessibilityLabel={label ?? `${count} unread`}
      accessible
      style={[styles.badge, { backgroundColor: theme.primary }]}
    >
      <Text style={{ color: theme.onPrimary, fontSize: 11, lineHeight: 14 }} variant="caption">
        {text}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    minWidth: 20,
    height: 20,
    paddingHorizontal: 6,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center"
  }
});
