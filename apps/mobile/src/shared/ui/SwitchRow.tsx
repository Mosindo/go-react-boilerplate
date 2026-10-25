import React from "react";
import { StyleSheet, Switch, View } from "react-native";
import { MIN_TOUCH_TARGET, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = {
  label: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function SwitchRow({ label, description, value, onValueChange, disabled, testID }: Props) {
  const { colors } = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.text}>
        <Text variant="label">{label}</Text>
        {description ? (
          <Text color="textMuted" variant="caption">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        disabled={disabled}
        onValueChange={onValueChange}
        testID={testID}
        thumbColor={colors.surface}
        trackColor={{ false: colors.border, true: colors.primary }}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { minHeight: MIN_TOUCH_TARGET + 8, flexDirection: "row", alignItems: "center", gap: spacing.md },
  text: { flex: 1, gap: 2 }
});
