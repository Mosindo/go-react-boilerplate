import React from "react";
import { StyleSheet, View } from "react-native";
import { APP_NAME } from "../../config";
import { spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export function BrandMark({ size = 56 }: { size?: number }) {
  const { colors } = useTheme();
  return (
    <View accessible accessibilityLabel={APP_NAME} accessibilityRole="header" style={styles.row}>
      <View
        style={[
          styles.halo,
          { width: size, height: size, borderRadius: size / 2, backgroundColor: colors.primarySoft }
        ]}
      >
        <View
          style={[
            styles.core,
            { width: size * 0.5, height: size * 0.5, borderRadius: size * 0.25, backgroundColor: colors.primary }
          ]}
        />
      </View>
      <Text style={{ fontSize: size * 0.6, lineHeight: size * 0.75 }} variant="title">
        {APP_NAME}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  halo: { alignItems: "center", justifyContent: "center" },
  core: {}
});
