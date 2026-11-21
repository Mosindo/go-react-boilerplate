import React from "react";
import { View } from "react-native";
import { makeStyles } from "../theme";
import { Text } from "./Text";

export function Section({ title, children, hint }: { title?: string; hint?: string; children: React.ReactNode }) {
  const styles = useStyles();
  return (
    <View style={styles.section}>
      {title ? (
        <Text variant="micro" tone="subtle" accessibilityRole="header">
          {title.toUpperCase()}
        </Text>
      ) : null}
      {children}
      {hint ? (
        <Text variant="caption" tone="subtle">
          {hint}
        </Text>
      ) : null}
    </View>
  );
}

export function Card({ children }: { children: React.ReactNode }) {
  const styles = useStyles();
  return <View style={styles.card}>{children}</View>;
}

const useStyles = makeStyles((t) => ({
  section: { gap: t.spacing.sm },
  card: {
    backgroundColor: t.colors.surface,
    borderRadius: t.radii.lg,
    borderWidth: 1,
    borderColor: t.colors.border,
    overflow: "hidden"
  }
}));
