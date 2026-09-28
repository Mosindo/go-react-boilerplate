import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "../ui/Text";

/** Titled group. `flush` removes inner padding so list rows can touch the edges. */
export function Section({ title, children, flush = false }: { title?: string; children: ReactNode; flush?: boolean }) {
  const theme = useTheme();
  return (
    <View style={styles.wrap}>
      {title ? (
        <Text tone="muted" variant="label">
          {title}
        </Text>
      ) : null}
      <View
        style={[
          styles.box,
          { backgroundColor: theme.surface, borderColor: theme.border },
          flush ? styles.flush : styles.padded
        ]}
      >
        {children}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.sm },
  box: { borderRadius: radius.lg, borderWidth: StyleSheet.hairlineWidth, overflow: "hidden" },
  padded: { padding: spacing.lg, gap: spacing.md },
  flush: { paddingVertical: spacing.xs }
});
