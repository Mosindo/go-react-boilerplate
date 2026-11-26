import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type HeaderProps = {
  title?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  large?: boolean;
  children?: React.ReactNode;
};

export function Header({ title, onBack, right, large, children }: HeaderProps) {
  return (
    <View style={[styles.row, large ? styles.large : null]}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} testID="header-back" /> : null}
      <View style={styles.center}>
        {children ??
          (title ? (
            <Text accessibilityRole="header" numberOfLines={1} variant={large ? "title" : "heading"}>
              {title}
            </Text>
          ) : null)}
      </View>
      {right ? <View style={styles.right}>{right}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs, minHeight: 56, paddingVertical: spacing.xs },
  large: { minHeight: 64 },
  center: { flex: 1 },
  right: { flexDirection: "row", alignItems: "center", gap: spacing.xxs }
});
