import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  icon?: IconName;
  onPress?: () => void;
  testID?: string;
};

export function Chip({ label, selected, highlighted, icon, onPress, testID }: Props) {
  const { colors } = useTheme();
  const active = selected || highlighted;
  const content = (
    <>
      {icon ? <Icon color={active ? "accent" : "textMuted"} name={icon} size={15} /> : null}
      <Text color={active ? "accent" : "text"} style={styles.label} variant="label">
        {label}
      </Text>
    </>
  );
  const style = [
    styles.chip,
    {
      backgroundColor: active ? colors.accentSoft : colors.surface,
      borderColor: active ? colors.accent : colors.border
    }
  ];
  if (!onPress) {
    return (
      <Pressable disabled style={style}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      onPress={onPress}
      style={({ pressed }) => [...style, pressed ? { opacity: 0.75 } : null]}
      testID={testID}
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
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm + 2,
    paddingVertical: 7
  },
  label: { fontWeight: "500" }
});
