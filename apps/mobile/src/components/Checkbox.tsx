import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text, colors, radii, spacing } from "../ui";

type CheckboxProps = {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  testID?: string;
};

export function Checkbox({ checked, onChange, label, testID }: CheckboxProps) {
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="checkbox"
      accessibilityState={{ checked }}
      onPress={() => onChange(!checked)}
      style={styles.row}
      testID={testID}
    >
      <View style={[styles.box, checked ? styles.checked : null]}>
        {checked ? <Ionicons color={colors.textOnAccent} name="checkmark" size={16} /> : null}
      </View>
      <Text style={styles.label} tone="muted" variant="caption">
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "flex-start",
    gap: spacing.sm,
    paddingVertical: spacing.xs
  },
  box: {
    width: 22,
    height: 22,
    borderRadius: radii.sm / 2,
    borderWidth: 1.5,
    borderColor: colors.borderStrong,
    alignItems: "center",
    justifyContent: "center",
    marginTop: 1
  },
  checked: {
    backgroundColor: colors.accent,
    borderColor: colors.accent
  },
  label: {
    flex: 1
  }
});
