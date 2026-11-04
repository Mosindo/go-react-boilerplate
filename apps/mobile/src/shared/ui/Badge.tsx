import React from "react";
import { View } from "react-native";
import { Text } from "./Text";
import { useColors } from "./theme";

export function Badge({ count }: { count: number }) {
  const c = useColors();
  if (count <= 0) return null;
  return (
    <View accessibilityLabel={`${count} non lu${count > 1 ? "s" : ""}`} style={{ minWidth: 20, height: 20, borderRadius: 10, paddingHorizontal: 6, backgroundColor: c.accent, alignItems: "center", justifyContent: "center" }}>
      <Text variant="caption" style={{ color: c.onAccent, fontWeight: "700" }}>{count > 99 ? "99+" : count}</Text>
    </View>
  );
}
