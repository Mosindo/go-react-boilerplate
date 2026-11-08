import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme/theme";
import { IconButton } from "../../ui/Button";
import { Text } from "../../ui/Text";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (v: number) => string;
  onChange: (v: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, format, onChange }: Props) {
  return (
    <View style={styles.row}>
      <Text style={styles.flex}>{label}</Text>
      <IconButton
        icon="remove-circle-outline"
        label={`Diminuer : ${label}`}
        onPress={() => onChange(Math.max(min, value - step))}
        disabled={value <= min}
      />
      <Text variant="heading" style={styles.value} accessibilityLiveRegion="polite">
        {format ? format(value) : value}
      </Text>
      <IconButton
        icon="add-circle-outline"
        label={`Augmenter : ${label}`}
        onPress={() => onChange(Math.min(max, value + step))}
        disabled={value >= max}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", paddingVertical: spacing.xs },
  flex: { flex: 1 },
  value: { minWidth: 72, textAlign: "center" }
});
