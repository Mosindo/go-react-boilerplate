import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme/tokens";
import { Text } from "../ui";

type Option<T extends string> = { value: T; label: string; description?: string };

type Props<T extends string> = {
  options: Option<T>[];
  selected: T[];
  onToggle: (value: T) => void;
  multiple?: boolean;
  testIDPrefix?: string;
};

/** Accessible radio/checkbox list with large touch targets. */
export function OptionList<T extends string>({ options, selected, onToggle, multiple, testIDPrefix }: Props<T>) {
  return (
    <View style={styles.list} accessibilityRole={multiple ? undefined : "radiogroup"}>
      {options.map((o) => {
        const isOn = selected.includes(o.value);
        return (
          <Pressable
            key={o.value}
            accessibilityRole={multiple ? "checkbox" : "radio"}
            accessibilityState={{ checked: isOn }}
            accessibilityLabel={o.label}
            onPress={() => onToggle(o.value)}
            testID={testIDPrefix ? `${testIDPrefix}-${o.value}` : undefined}
            style={({ pressed }) => [styles.row, isOn && styles.rowOn, pressed && { opacity: 0.8 }]}
          >
            <View style={styles.texts}>
              <Text variant="bodyStrong">{o.label}</Text>
              {o.description ? (
                <Text variant="caption" tone="muted">
                  {o.description}
                </Text>
              ) : null}
            </View>
            <Ionicons
              name={isOn ? (multiple ? "checkbox" : "radio-button-on") : multiple ? "square-outline" : "radio-button-off"}
              size={22}
              color={isOn ? colors.accent : colors.textSubtle}
            />
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.lg,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    minHeight: 56
  },
  rowOn: { borderColor: colors.accent, backgroundColor: colors.accentSoft },
  texts: { flex: 1, gap: 2 }
});
