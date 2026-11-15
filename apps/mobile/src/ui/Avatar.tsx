import React from "react";
import { StyleSheet, View } from "react-native";
import type { Photo } from "../api/types";
import { useTheme } from "../theme";
import { PhotoImage } from "./PhotoImage";
import { Text } from "./Text";

export function Avatar({ photo, name, size = 52, ring }: { photo?: Photo; name: string; size?: number; ring?: boolean }) {
  const { colors } = useTheme();
  const frame = {
    width: size,
    height: size,
    borderRadius: size / 2,
    borderWidth: ring ? 2 : 0,
    borderColor: colors.primary
  };
  if (photo) {
    return (
      <View style={[frame, styles.clip]}>
        <PhotoImage photo={photo} style={{ width: "100%", height: "100%" }} />
      </View>
    );
  }
  return (
    <View style={[frame, styles.center, { backgroundColor: colors.primarySoft }]}>
      <Text variant="heading" color="primary">
        {name.slice(0, 1).toUpperCase() || "?"}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  clip: { overflow: "hidden" },
  center: { alignItems: "center", justifyContent: "center" }
});
