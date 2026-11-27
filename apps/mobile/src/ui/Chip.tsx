import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/palette";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  icon?: IconName;
  highlighted?: boolean;
  testID?: string;
};

export function Chip({ label, selected, onPress, icon, highlighted, testID }: ChipProps) {
  const { colors } = useTheme();
  const active = selected || highlighted;
  const content = (
    <>
      {icon ? <Icon name={icon} size={15} color={active ? colors.primary : colors.textMuted} /> : null}
      <Text variant="label" style={{ color: active ? colors.primary : colors.text }}>
        {label}
      </Text>
    </>
  );
  const style = [
    styles.chip,
    {
      backgroundColor: active ? colors.primarySoft : colors.surface,
      borderColor: active ? colors.primary : colors.border
    }
  ];
  if (!onPress) {
    return (
      <Pressable disabled style={style} testID={testID}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [...style, { opacity: pressed ? 0.8 : 1 }]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: spacing.sm + 2,
    paddingVertical: 8,
    borderRadius: radii.pill,
    borderWidth: 1
  }
});
