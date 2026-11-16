import React from "react";
import { StyleSheet, View } from "react-native";
import { colors, spacing } from "../theme/tokens";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

export type HeaderProps = {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  large?: boolean;
};

export function Header({ title, subtitle, onBack, right, large }: HeaderProps) {
  return (
    <View style={[styles.row, large && styles.large]}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} size={40} background={colors.transparent} /> : null}
      <View style={styles.titles}>
        <Text variant={large ? "title" : "heading"} numberOfLines={1}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" tone="muted" numberOfLines={1}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ? <View style={styles.right}>{right}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, minHeight: 52, marginBottom: spacing.md },
  large: { marginBottom: spacing.lg },
  titles: { flex: 1, gap: 2 },
  right: { flexDirection: "row", alignItems: "center", gap: spacing.sm }
});
