import React from "react";
import { StyleSheet, View } from "react-native";
import type { Photo } from "../../api/types";
import { useTheme } from "../../theme";
import { PhotoImage } from "./PhotoImage";
import { Text } from "./Text";

export function Avatar({ name, photo, size = 48 }: { name: string; photo?: Photo | null; size?: number }) {
  const theme = useTheme();
  const initial = name.trim().charAt(0).toUpperCase() || "?";
  return (
    <View
      accessibilityIgnoresInvertColors
      style={[styles.base, { width: size, height: size, borderRadius: size / 2, backgroundColor: theme.surfaceAlt }]}
    >
      {photo ? (
        <PhotoImage accessibilityLabel={`${name}'s photo`} photo={photo} style={{ width: size, height: size }} />
      ) : (
        <Text style={{ fontSize: size * 0.4 }} tone="muted" variant="heading">
          {initial}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({ base: { overflow: "hidden", alignItems: "center", justifyContent: "center" } });
