import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  title?: string;
  subtitle?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  large?: boolean;
};

export function Header({ title, subtitle, onBack, right, large }: Props) {
  return (
    <View style={[styles.row, large && styles.large]}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} size={40} background="surface" /> : null}
      <View style={styles.titles}>
        {title ? (
          <Text variant={large ? "title" : "heading"} accessibilityRole="header" numberOfLines={1}>
            {title}
          </Text>
        ) : null}
        {subtitle ? (
          <Text variant="caption" color="textMuted" numberOfLines={1}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ? <View style={styles.right}>{right}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.sm, minHeight: 56 },
  large: { paddingTop: spacing.md, paddingBottom: spacing.md },
  titles: { flex: 1 },
  right: { flexDirection: "row", alignItems: "center", gap: spacing.xs }
});
