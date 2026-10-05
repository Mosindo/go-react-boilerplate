import type { ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Text } from "../ui/Text";

export function Section({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <View style={styles.section}>
      <Text variant="heading">{title}</Text>
      {description ? (
        <Text variant="caption" tone="muted">
          {description}
        </Text>
      ) : null}
      <View style={styles.body}>{children}</View>
    </View>
  );
}

const styles = StyleSheet.create({ section: { gap: spacing.xs }, body: { gap: spacing.md, marginTop: spacing.sm } });
