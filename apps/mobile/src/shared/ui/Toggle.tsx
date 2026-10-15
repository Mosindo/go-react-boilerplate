import React from "react";
import { Switch, View } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { spacing } from "./tokens";

type ToggleProps = {
  label: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function Toggle({ description, disabled, label, onValueChange, testID, value }: ToggleProps) {
  const palette = useTheme();
  return (
    <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md, minHeight: 56 }}>
      <View style={{ flex: 1, gap: 2 }}>
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
        onValueChange={onValueChange}
        thumbColor="#FFFFFF"
        trackColor={{ false: palette.borderStrong, true: palette.primary }}
        testID={testID}
        value={value}
      />
    </View>
  );
}
