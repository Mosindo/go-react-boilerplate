import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme";
import { Chip, Text } from "../ui";

type Props<T extends string> = {
  label: string;
  options: T[];
  labels: Record<T, string>;
  value: T[];
  onChange: (value: T[]) => void;
  multiple?: boolean;
  testIDPrefix?: string;
};

export function ChoiceGroup<T extends string>({ label, options, labels, value, onChange, multiple, testIDPrefix }: Props<T>) {
  const toggle = (option: T) => {
    if (!multiple) {
      onChange([option]);
      return;
    }
    onChange(value.includes(option) ? value.filter((v) => v !== option) : [...value, option]);
  };
  return (
    <View style={styles.group}>
      <Text color="textMuted" variant="label">
        {label}
      </Text>
      <View accessibilityRole={multiple ? undefined : "radiogroup"} style={styles.row}>
        {options.map((option) => (
          <Chip
            key={option}
            label={labels[option]}
            onPress={() => toggle(option)}
            selected={value.includes(option)}
            testID={testIDPrefix ? `${testIDPrefix}-${option}` : undefined}
          />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  group: { gap: spacing.xs },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
