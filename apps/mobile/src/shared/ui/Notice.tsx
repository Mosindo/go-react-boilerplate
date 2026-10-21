import React from "react";
import { StyleSheet, View } from "react-native";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = { message: string; tone?: "error" | "info" | "success"; testID?: string };

/** Inline banner for form-level feedback. */
export function Notice({ message, tone = "error", testID }: Props) {
  const theme = useTheme();
  const color = tone === "error" ? theme.danger : tone === "success" ? theme.success : theme.accent;
  return (
    <View testID={testID} accessibilityRole="alert" style={[styles.box, { borderColor: color, backgroundColor: theme.surface }]}>
      <Text variant="caption" style={{ color }}>
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: { borderWidth: 1, borderRadius: radius.md, padding: spacing.md }
});
