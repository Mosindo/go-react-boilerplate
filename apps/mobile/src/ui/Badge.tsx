import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../theme";
import { Text } from "./Text";

export function Badge({ count, testID }: { count: number; testID?: string }) {
  const { colors } = useTheme();
  if (count <= 0) return null;
  return (
    <View accessibilityLabel={`${count} non lus`} style={[styles.badge, { backgroundColor: colors.primary }]} testID={testID}>
      <Text style={[styles.text, { color: colors.onPrimary }]} variant="overline">
        {count > 99 ? "99+" : String(count)}
      </Text>
    </View>
  );
}

export function Dot() {
  const { colors } = useTheme();
  return <View style={[styles.dot, { backgroundColor: colors.primary }]} />;
}

const styles = StyleSheet.create({
  badge: { minWidth: 20, height: 20, borderRadius: 10, paddingHorizontal: 6, alignItems: "center", justifyContent: "center" },
  text: { fontSize: 11, lineHeight: 14, letterSpacing: 0 },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
