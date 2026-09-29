import React from "react";
import { View } from "react-native";
import { IconButton } from "./IconButton";
import { Text } from "./Text";
import { controls } from "./tokens";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type StepperProps = {
  label: string;
  valueLabel: string;
  onDecrement: () => void;
  onIncrement: () => void;
  canDecrement?: boolean;
  canIncrement?: boolean;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  row: {
    flexDirection: "row" as const,
    alignItems: "center" as const,
    justifyContent: "space-between" as const,
    minHeight: controls.minTarget,
    gap: t.spacing.md
  },
  controls: { flexDirection: "row" as const, alignItems: "center" as const, gap: t.spacing.sm },
  value: { minWidth: 72, textAlign: "center" as const }
});

/** "- value +" control used where a slider would be (age range, distance). */
export function Stepper({
  canDecrement = true,
  canIncrement = true,
  label,
  onDecrement,
  onIncrement,
  testID,
  valueLabel
}: StepperProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View
      accessibilityLabel={`${label}: ${valueLabel}`}
      accessibilityRole="adjustable"
      accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
      onAccessibilityAction={(event) => {
        if (event.nativeEvent.actionName === "increment" && canIncrement) {
          onIncrement();
        } else if (event.nativeEvent.actionName === "decrement" && canDecrement) {
          onDecrement();
        }
      }}
      style={styles.row}
      testID={testID}
    >
      <Text weight="semibold">{label}</Text>
      <View style={styles.controls}>
        <IconButton
          accessibilityLabel={`Decrease ${label}`}
          disabled={!canDecrement}
          filled
          glyph="−"
          onPress={onDecrement}
        />
        <Text style={styles.value} weight="bold">
          {valueLabel}
        </Text>
        <IconButton
          accessibilityLabel={`Increase ${label}`}
          disabled={!canIncrement}
          filled
          glyph="+"
          onPress={onIncrement}
        />
      </View>
    </View>
  );
}
