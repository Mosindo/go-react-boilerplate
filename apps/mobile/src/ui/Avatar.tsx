import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { RemoteImage } from "./RemoteImage";
import { Text } from "./Text";

export function Avatar({ path, name, size = 52, ring }: { path?: string | null; name: string; size?: number; ring?: boolean }) {
  const { colors } = useTheme();
  const frame = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View style={[frame, ring ? { borderWidth: 2.5, borderColor: colors.accent, padding: 2 } : null]}>
      {path ? (
        <RemoteImage
          path={path}
          style={[StyleSheet.absoluteFill, frame, ring ? { margin: 2.5 } : null]}
          accessibilityLabel={`Photo de ${name}`}
        />
      ) : (
        <View style={[styles.fallback, frame, { backgroundColor: colors.primarySoft }]}>
          <Text variant="heading" tone="primary">
            {name.slice(0, 1).toUpperCase()}
          </Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  fallback: { alignItems: "center", justifyContent: "center" }
});
