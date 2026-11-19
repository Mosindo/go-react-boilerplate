import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme/tokens";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  title?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  large?: boolean;
};

export function Header({ title, onBack, right, large }: Props) {
  return (
    <View style={styles.row}>
      <View style={styles.side}>{onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} /> : null}</View>
      {title ? (
        <Text variant={large ? "title" : "heading"} numberOfLines={1} style={styles.title} accessibilityRole="header">
          {title}
        </Text>
      ) : (
        <View style={styles.title} />
      )}
      <View style={[styles.side, styles.right]}>{right}</View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", minHeight: 56, gap: spacing.xs },
  side: { minWidth: 44, flexDirection: "row", alignItems: "center" },
  right: { justifyContent: "flex-end", gap: spacing.xs },
  title: { flex: 1 }
});
