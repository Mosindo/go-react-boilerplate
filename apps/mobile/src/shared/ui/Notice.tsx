import React from "react";
import { StyleSheet, View } from "react-native";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type NoticeProps = { message: string; kind?: "info" | "error" | "success" };

/** Inline banner for form-level information or errors. */
export function Notice({ message, kind = "info" }: NoticeProps) {
  const theme = useTheme();
  const accent = kind === "error" ? theme.danger : kind === "success" ? theme.success : theme.accent;
  return (
    <View
      accessibilityLiveRegion="polite"
      accessibilityRole={kind === "error" ? "alert" : undefined}
      style={[styles.box, { borderColor: accent, backgroundColor: theme.surface }]}
    >
      <Text style={{ color: theme.text }} variant="label">
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: { borderWidth: 1, borderLeftWidth: 4, borderRadius: radius.md, padding: spacing.md }
});
