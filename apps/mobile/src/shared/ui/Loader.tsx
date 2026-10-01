import React from "react";
import { ActivityIndicator, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "./theme";

type LoaderProps = { size?: "small" | "large"; style?: StyleProp<ViewStyle> };

export function Loader({ size = "small", style }: LoaderProps) {
  const { colors } = useTheme();
  return (
    <View accessibilityRole="progressbar" style={[styles.wrap, style]}>
      <ActivityIndicator color={colors.primary} size={size} />
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { alignItems: "center", justifyContent: "center" } });
