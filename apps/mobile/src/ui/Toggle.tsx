import React from "react";
import { StyleSheet, Switch, View } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./theme";

type ToggleProps = {
  label: string;
  description?: string;
  value: boolean;
  onChange: (next: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function Toggle({ label, description, value, onChange, disabled, testID }: ToggleProps) {
  return (
    <View style={styles.row}>
      <View style={styles.body}>
        <Text variant="label">{label}</Text>
        {description ? (
          <Text tone="muted" variant="caption">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        disabled={disabled}
        onValueChange={onChange}
        testID={testID}
        thumbColor={colors.text}
        trackColor={{ false: colors.border, true: colors.accent }}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surface
  },
  body: {
    flex: 1,
    gap: 2
  }
});
