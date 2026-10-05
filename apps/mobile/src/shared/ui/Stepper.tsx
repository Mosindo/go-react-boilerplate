import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

/** − value + control with bounds; used for age and distance preferences (no slider dependency needed). */
export function Stepper({ label, value, min, max, step = 1, format, onChange }: { label: string; value: number; min: number; max: number; step?: number; format?: (value: number) => string; onChange: (value: number) => void }) {
  return (
    <View style={styles.row} accessibilityRole="adjustable" accessibilityLabel={label} accessibilityValue={{ min, max, now: value, text: format?.(value) ?? String(value) }}>
      <Text variant="bodyStrong" style={styles.label}>
        {label}
      </Text>
      <IconButton icon="remove-circle-outline" label={`Diminuer ${label}`} onPress={() => onChange(Math.max(min, value - step))} />
      <Text variant="bodyStrong" center style={styles.value}>
        {format?.(value) ?? value}
      </Text>
      <IconButton icon="add-circle-outline" label={`Augmenter ${label}`} onPress={() => onChange(Math.min(max, value + step))} />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  label: { flex: 1 },
  value: { minWidth: 76 }
});
