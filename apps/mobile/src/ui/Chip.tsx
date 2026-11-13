import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  highlighted?: boolean;
  testID?: string;
};

export function Chip({ label, selected, onPress, highlighted, testID }: ChipProps) {
  const { colors, radii } = useTheme();
  const bg = selected ? colors.primary : highlighted ? colors.accentSoft : colors.surface;
  const fg = selected ? colors.onPrimary : highlighted ? colors.accent : colors.text;
  const content = (
    <View style={styles.row}>
      {selected ? <Ionicons name="checkmark" size={14} color={fg} /> : null}
      <Text variant="label" weight="medium" style={{ color: fg }}>
        {label}
      </Text>
    </View>
  );
  const style = [
    styles.chip,
    { backgroundColor: bg, borderRadius: radii.pill, borderColor: selected ? colors.primary : colors.border }
  ];
  if (!onPress) {
    return <View style={style}>{content}</View>;
  }
  return (
    <Pressable
      testID={testID}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [...style, { opacity: pressed ? 0.75 : 1 }]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: { paddingHorizontal: 14, paddingVertical: 8, borderWidth: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: 6 }
});
