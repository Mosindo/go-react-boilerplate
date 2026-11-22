import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { IconButton } from "./IconButton";
import { Text } from "./Text";
import { spacing } from "./theme";

type HeaderProps = {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  right?: ReactNode;
};

export function Header({ title, subtitle, onBack, right }: HeaderProps) {
  return (
    <View style={styles.row}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} size={40} /> : null}
      <View style={styles.titles}>
        <Text accessibilityRole="header" numberOfLines={1} variant={onBack ? "heading" : "title"}>
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ? <View style={styles.right}>{right}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingBottom: spacing.md
  },
  titles: {
    flex: 1,
    gap: 2
  },
  right: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  }
});
