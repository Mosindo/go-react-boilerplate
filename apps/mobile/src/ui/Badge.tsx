import React from "react";
import { StyleSheet, View } from "react-native";
import { colors, radii } from "../theme/tokens";
import { Text } from "./Text";

export function Badge({ count, testID }: { count: number; testID?: string }) {
  if (count <= 0) {
    return null;
  }
  return (
    <View style={styles.badge} accessibilityLabel={`${count} non lu${count > 1 ? "s" : ""}`} testID={testID}>
      <Text variant="label" style={styles.text}>
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
    backgroundColor: colors.accent,
    alignItems: "center",
    justifyContent: "center"
  },
  text: { color: colors.textOnAccent, letterSpacing: 0 }
});
