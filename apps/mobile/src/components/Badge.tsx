import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../theme";
import { Text } from "./Text";

export function Badge({ count, testID }: { count: number; testID?: string }) {
  const { colors } = useTheme();
  if (count <= 0) return null;
  return (
    <View style={[styles.badge, { backgroundColor: colors.accent }]} testID={testID} accessibilityLabel={`${count} unread`}>
      <Text variant="caption" style={styles.text}>
        {count > 99 ? "99+" : String(count)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: { minWidth: 20, height: 20, borderRadius: 10, paddingHorizontal: 5, alignItems: "center", justifyContent: "center" },
  text: { color: "#fff", fontWeight: "700", fontSize: 11, lineHeight: 14 }
});
