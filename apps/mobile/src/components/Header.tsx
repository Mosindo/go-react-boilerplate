import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { colors, spacing } from "../theme/tokens";
import { AppText } from "./AppText";
import { IconButton } from "./IconButton";

type Props = { title: string; subtitle?: string; onBack?: () => void; right?: ReactNode };

export function Header({ title, subtitle, onBack, right }: Props) {
  return (
    <View style={styles.row}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} tone="plain" testID="header-back" /> : null}
      <View style={styles.titles}>
        <AppText variant={onBack ? "heading" : "title"} accessibilityRole="header" numberOfLines={1}>
          {title}
        </AppText>
        {subtitle ? (
          <AppText variant="caption" color={colors.textMuted} numberOfLines={1}>
            {subtitle}
          </AppText>
        ) : null}
      </View>
      {right}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm, minHeight: 56 },
  titles: { flex: 1 }
});
