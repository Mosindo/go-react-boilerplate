import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
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
    <View style={styles.row}>
      {icon ? <Icon name={icon} size={14} color={active ? "primary" : "textMuted"} /> : null}
      <Text variant="label" color={active ? "primary" : "text"} style={styles.label}>
        {label}
      </Text>
    </View>
  );
  const style = [
    styles.base,
    {
      backgroundColor: active ? colors.primarySoft : colors.surfaceAlt,
      borderColor: selected ? colors.primary : active ? colors.primarySoft : colors.border
    }
  ];
  if (!onPress) return <View style={style}>{content}</View>;
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: Boolean(selected) }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [...style, pressed && { opacity: 0.8 }]}
    >
      {content}
    </Pressable>
  );
}

export function ChipGroup({ children }: { children: React.ReactNode }) {
  return <View style={styles.group}>{children}</View>;
}

const styles = StyleSheet.create({
  base: { borderRadius: radii.pill, borderWidth: 1, paddingHorizontal: spacing.sm, paddingVertical: 7 },
  row: { flexDirection: "row", alignItems: "center", gap: 5 },
  label: { fontSize: 13 },
  group: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
