import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text, colors, radii, spacing } from "../ui";

type Option<T extends string> = { value: T; label: string; description?: string };

type ChoiceListProps<T extends string> = {
  options: Option<T>[];
  selected: T[];
  onToggle: (value: T) => void;
  multiple?: boolean;
  testIDPrefix?: string;
};

/** Large, accessible single/multi choice rows. */
export function ChoiceList<T extends string>({ options, selected, onToggle, multiple, testIDPrefix }: ChoiceListProps<T>) {
  return (
    <View style={styles.list}>
      {options.map((o) => {
        const isOn = selected.includes(o.value);
        return (
          <Pressable
            accessibilityLabel={o.label}
            accessibilityRole={multiple ? "checkbox" : "radio"}
            accessibilityState={multiple ? { checked: isOn } : { selected: isOn }}
            key={o.value}
            onPress={() => onToggle(o.value)}
            style={({ pressed }) => [styles.row, isOn ? styles.on : null, pressed ? styles.pressed : null]}
            testID={testIDPrefix ? `${testIDPrefix}-${o.value}` : undefined}
          >
            <View style={styles.body}>
              <Text variant="label">{o.label}</Text>
              {o.description ? (
                <Text tone="muted" variant="caption">
                  {o.description}
                </Text>
              ) : null}
            </View>
            <Ionicons
              color={isOn ? colors.accent : colors.textSubtle}
              name={isOn ? (multiple ? "checkbox" : "radio-button-on") : multiple ? "square-outline" : "radio-button-off"}
              size={22}
            />
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: spacing.xs
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    minHeight: 60,
    paddingHorizontal: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  on: {
    borderColor: colors.accent,
    backgroundColor: colors.accentSoft
  },
  pressed: {
    opacity: 0.85
  },
  body: {
    flex: 1,
    gap: 2
  }
});
