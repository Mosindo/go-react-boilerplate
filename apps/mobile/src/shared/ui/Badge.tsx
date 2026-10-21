import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";

/** Round counter; renders nothing for 0. */
export function Badge({ count, testID }: { count: number; testID?: string }) {
  const theme = useTheme();
  if (count <= 0) return null;
  return (
    <View
      testID={testID}
      accessibilityLabel={`${count} non lu${count > 1 ? "s" : ""}`}
      style={[styles.badge, { backgroundColor: theme.primary }]}
    >
      <Text variant="caption" style={{ color: theme.primaryText, fontWeight: "700", fontSize: 11, lineHeight: 14 }}>
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: { minWidth: 20, height: 20, borderRadius: 10, paddingHorizontal: 5, alignItems: "center", justifyContent: "center" }
});
