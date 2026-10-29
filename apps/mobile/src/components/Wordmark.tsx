import React from "react";
import { StyleSheet, Text } from "react-native";
import { useTheme } from "../theme/ThemeProvider";

export function Wordmark({ size = 40 }: { size?: number }) {
  const theme = useTheme();
  return (
    <Text
      accessibilityRole="header"
      style={[styles.text, { color: theme.primary, fontSize: size }]}
    >
      sillage
    </Text>
  );
}

const styles = StyleSheet.create({
  text: { fontWeight: "300", letterSpacing: -1.5, fontStyle: "italic" }
});
