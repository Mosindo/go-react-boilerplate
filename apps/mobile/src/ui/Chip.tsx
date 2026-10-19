import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { MIN_TOUCH, radius, spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; testID?: string };

export function Chip({ label, selected, onPress, testID }: Props) {
  const { colors } = useTheme();
  const body = (
    <Text variant="caption" bold tone={selected ? "accent" : "default"}>
      {label}
    </Text>
  );
  const style = [
    styles.chip,
    {
      backgroundColor: selected ? colors.accentSoft : colors.surfaceAlt,
      borderColor: selected ? colors.accent : "transparent"
    }
  ];
  if (!onPress)
    return (
      <Pressable style={style} disabled accessibilityRole="text">
        {body}
      </Pressable>
    );
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityState={{ selected: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      style={[style, styles.touch]}
    >
      {body}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderRadius: radius.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2
  },
  touch: { minHeight: MIN_TOUCH - 8, justifyContent: "center" }
});
