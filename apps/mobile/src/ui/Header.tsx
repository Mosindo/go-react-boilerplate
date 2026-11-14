import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme/palette";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  large?: boolean;
};

export function Header({ title, subtitle, onBack, right, large }: Props) {
  return (
    <View style={styles.row}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} /> : null}
      <View style={styles.titles}>
        <Text accessibilityRole="header" numberOfLines={1} variant={large ? "display" : "heading"}>
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs, minHeight: 52, marginBottom: spacing.xs },
  titles: { flex: 1, gap: 2 }
});
