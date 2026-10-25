import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { MIN_TOUCH_TARGET, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  /** Static chips (no onPress) are used to display interests. */
  onPress?: () => void;
  highlighted?: boolean;
  disabled?: boolean;
  testID?: string;
};

export const Chip = React.memo(function Chip({
  label,
  selected = false,
  onPress,
  highlighted = false,
  disabled,
  testID
}: Props) {
  const { colors } = useTheme();
  const active = selected || highlighted;
  const content = (
    <Text style={{ color: active ? colors.onPrimary : colors.text }} variant="caption">
      {label}
    </Text>
  );
  const chipStyle = [
    styles.chip,
    {
      backgroundColor: active ? colors.primary : colors.surface,
      borderColor: active ? colors.primary : colors.border,
      opacity: disabled ? 0.45 : 1
    }
  ];

  if (!onPress) {
    return (
      <Pressable
        accessible
        accessibilityLabel={highlighted ? `${label}, en commun` : label}
        disabled
        style={chipStyle}
        testID={testID}
      >
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled: Boolean(disabled) }}
      disabled={disabled}
      onPress={onPress}
      style={[chipStyle, styles.interactive]}
      testID={testID}
    >
      {content}
    </Pressable>
  );
});

const styles = StyleSheet.create({
  chip: {
    borderRadius: radius.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2,
    justifyContent: "center"
  },
  interactive: { minHeight: MIN_TOUCH_TARGET - 8 }
});
