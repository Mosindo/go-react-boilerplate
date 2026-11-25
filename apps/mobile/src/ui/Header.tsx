import React from "react";
import { StyleSheet, View } from "react-native";
import { colors, spacing } from "../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type HeaderProps = {
  title?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  large?: boolean;
};

export function Header({ title, onBack, right, large }: HeaderProps) {
  return (
    <View style={styles.row}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} testID="header-back" /> : null}
      <View style={styles.title}>
        {title ? (
          <Text variant={large ? "title" : "heading"} numberOfLines={1} accessibilityRole="header">
            {title}
          </Text>
        ) : null}
      </View>
      {right ? <View style={styles.right}>{right}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    minHeight: 52,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingBottom: spacing.xs,
    borderBottomColor: colors.border
  },
  title: {
    flex: 1
  },
  right: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xxs
  }
});
