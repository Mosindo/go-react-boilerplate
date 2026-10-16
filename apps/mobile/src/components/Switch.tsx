import React from "react";
import { StyleSheet, Switch as RNSwitch, View } from "react-native";
import { Text, colors, spacing } from "../shared/ui";

type ToggleRowProps = {
  label: string;
  description?: string;
  value: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function ToggleRow({ label, description, value, onChange, disabled, testID }: ToggleRowProps) {
  return (
    <View style={styles.row}>
      <View style={styles.copy}>
        <Text variant="label" weight="bold">
          {label}
        </Text>
        {description ? (
          <Text tone="muted" variant="caption">
            {description}
          </Text>
        ) : null}
      </View>
      <RNSwitch
        accessibilityLabel={label}
        disabled={disabled}
        ios_backgroundColor={colors.surfaceSubtle}
        onValueChange={onChange}
        testID={testID}
        thumbColor={colors.inverse}
        trackColor={{ false: colors.borderStrong, true: colors.primary }}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  copy: { flex: 1, gap: spacing.xxs }
});
