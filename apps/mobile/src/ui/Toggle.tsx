import React from "react";
import { StyleSheet, Switch, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { Text } from "./Text";

type Props = {
  label: string;
  description?: string;
  value: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function Toggle({ label, description, value, onChange, disabled, testID }: Props) {
  const { colors } = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.text}>
        <Text variant="bodyStrong">{label}</Text>
        {description ? (
          <Text color="textMuted" variant="caption">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        disabled={disabled}
        onValueChange={onChange}
        testID={testID}
        thumbColor={colors.surface}
        trackColor={{ false: colors.borderStrong, true: colors.accent }}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.xs },
  text: { flex: 1, gap: 2 }
});
