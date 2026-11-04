import React from "react";
import { View } from "react-native";
import { RemoteImage } from "./RemoteImage";
import { Text } from "./Text";
import { useColors } from "./theme";

export function Avatar({ path, name, size = 52 }: { path: string | null | undefined; name: string; size?: number }) {
  const c = useColors();
  if (path) return <RemoteImage path={path} accessibilityLabel={`Photo de ${name}`} style={{ width: size, height: size, borderRadius: size / 2 }} />;
  return (
    <View accessibilityLabel={name} style={{ width: size, height: size, borderRadius: size / 2, backgroundColor: c.surfaceMuted, alignItems: "center", justifyContent: "center" }}>
      <Text variant="label" tone="primary" style={{ fontSize: size / 3 }}>{name.slice(0, 1).toUpperCase()}</Text>
    </View>
  );
}
