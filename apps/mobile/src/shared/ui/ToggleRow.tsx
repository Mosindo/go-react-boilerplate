import React from "react";
import { StyleSheet, Switch, View } from "react-native";
import { Text } from "./Text";
import { colors, spacing } from "./tokens";

type ToggleRowProps = {
  title: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function ToggleRow({ description, disabled, onValueChange, testID, title, value }: ToggleRowProps) {
  return (
    <View style={styles.row}>
      <View style={styles.texts}>
        <Text weight="medium">{title}</Text>
        {description ? (
          <Text tone="muted" variant="caption">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={title}
        disabled={disabled}
        onValueChange={onValueChange}
        testID={testID}
        thumbColor={value ? colors.primary : colors.textMuted}
        trackColor={{ false: colors.border, true: colors.primarySoft }}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md
  },
  texts: {
    flex: 1,
    gap: 2
  }
});
