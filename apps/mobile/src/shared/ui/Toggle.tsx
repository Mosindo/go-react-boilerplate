import React from "react";
import { StyleSheet, Switch, View } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { spacing } from "./tokens";

type ToggleProps = {
  label: string;
  description?: string;
  value: boolean;
  disabled?: boolean;
  onChange: (value: boolean) => void;
};

export function Toggle({ label, description, value, disabled, onChange }: ToggleProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.copy}>
        <Text weight="semibold">{label}</Text>
        {description ? (
          <Text tone="muted" variant="label">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        disabled={disabled}
        onValueChange={onChange}
        thumbColor="#FFFFFF"
        trackColor={{ false: colors.border, true: colors.primary }}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  copy: { flex: 1, gap: 2 }
});
