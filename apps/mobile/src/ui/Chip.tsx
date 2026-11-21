import React from "react";
import { Pressable } from "react-native";
import { makeStyles } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  icon?: IconName;
  disabled?: boolean;
  testID?: string;
};

export function Chip({ label, selected = false, onPress, icon, disabled, testID }: ChipProps) {
  const styles = useStyles();
  const interactive = Boolean(onPress);
  return (
    <Pressable
      testID={testID}
      accessibilityRole={interactive ? "checkbox" : "text"}
      accessibilityState={interactive ? { checked: selected, disabled } : undefined}
      accessibilityLabel={label}
      disabled={!interactive || disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.chip,
        selected ? styles.selected : null,
        pressed ? styles.pressed : null,
        disabled ? styles.disabled : null
      ]}
    >
      {icon ? <Icon name={icon} size={15} color={selected ? "textOnPrimary" : "textMuted"} /> : null}
      <Text variant="label" tone={selected ? "onPrimary" : "default"}>
        {label}
      </Text>
    </Pressable>
  );
}

const useStyles = makeStyles((t) => ({
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: t.spacing.xs,
    paddingHorizontal: t.spacing.md,
    paddingVertical: t.spacing.sm,
    borderRadius: t.radii.pill,
    borderWidth: 1,
    borderColor: t.colors.border,
    backgroundColor: t.colors.surface
  },
  selected: { backgroundColor: t.colors.primary, borderColor: t.colors.primary },
  pressed: { opacity: 0.75 },
  disabled: { opacity: 0.45 }
}));
