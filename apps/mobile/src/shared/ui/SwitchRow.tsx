import React from "react";
import { Switch, View } from "react-native";
import { Text } from "./Text";
import { controls } from "./tokens";
import { useTheme, type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type SwitchRowProps = {
  title: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  row: {
    flexDirection: "row" as const,
    alignItems: "center" as const,
    justifyContent: "space-between" as const,
    gap: t.spacing.md,
    minHeight: controls.minTarget
  },
  copy: { flex: 1, gap: t.spacing.xxs }
});

export function SwitchRow({ description, disabled, onValueChange, testID, title, value }: SwitchRowProps) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={styles.row}>
      <View style={styles.copy}>
        <Text weight="semibold">{title}</Text>
        {description ? (
          <Text tone="muted" variant="caption">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={title}
        disabled={disabled}
        ios_backgroundColor={theme.colors.borderStrong}
        onValueChange={onValueChange}
        testID={testID}
        thumbColor={theme.colors.surface}
        trackColor={{ false: theme.colors.borderStrong, true: theme.colors.primary }}
        value={value}
      />
    </View>
  );
}
