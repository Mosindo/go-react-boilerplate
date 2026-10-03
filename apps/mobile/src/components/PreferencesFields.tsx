import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { GENDER_LABELS, clamp } from "../lib/format";
import { Text, colors, radii, spacing } from "../shared/ui";
import { Chip } from "./Chip";

export const defaultPreferences: Preferences = {
  interestedIn: ["man", "woman", "nonbinary"],
  minAge: 18,
  maxAge: 60,
  maxDistanceKm: 50
};

const DISTANCE_STEPS = [5, 10, 25, 50, 100, 250, null] as const;

function Stepper({ label, onChange, value, min, max, testID }: { label: string; value: number; min: number; max: number; onChange: (v: number) => void; testID: string }) {
  return (
    <View style={styles.stepper}>
      <Text weight="semibold">{label}</Text>
      <View style={styles.stepperControls}>
        <Pressable accessibilityLabel={`Decrease ${label}`} accessibilityRole="button" onPress={() => onChange(clamp(value - 1, min, max))} style={styles.stepButton} testID={`${testID}-minus`}>
          <Text weight="bold">−</Text>
        </Pressable>
        <Text style={styles.stepValue} testID={`${testID}-value`} weight="bold">
          {value}
        </Text>
        <Pressable accessibilityLabel={`Increase ${label}`} accessibilityRole="button" onPress={() => onChange(clamp(value + 1, min, max))} style={styles.stepButton} testID={`${testID}-plus`}>
          <Text weight="bold">+</Text>
        </Pressable>
      </View>
    </View>
  );
}

type Props = { value: Preferences; onChange: (next: Preferences) => void };

export function PreferencesFields({ onChange, value }: Props) {
  const toggleGender = (g: Gender) => {
    const has = value.interestedIn.includes(g);
    if (has && value.interestedIn.length === 1) {
      return; // at least one option stays selected
    }
    onChange({ ...value, interestedIn: has ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g] });
  };
  const distanceIndex = DISTANCE_STEPS.findIndex((d) => d === value.maxDistanceKm);

  return (
    <View style={styles.stack}>
      <View style={styles.field}>
        <Text variant="label" weight="semibold">
          Show me
        </Text>
        <View style={styles.wrap}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip key={g} label={GENDER_LABELS[g]} onPress={() => toggleGender(g)} selected={value.interestedIn.includes(g)} testID={`pref-gender-${g}`} />
          ))}
        </View>
      </View>
      <View style={styles.field}>
        <Text variant="label" weight="semibold">
          Age range
        </Text>
        <Stepper
          label="Minimum age"
          max={value.maxAge}
          min={18}
          onChange={(v) => onChange({ ...value, minAge: v })}
          testID="pref-min-age"
          value={value.minAge}
        />
        <Stepper
          label="Maximum age"
          max={99}
          min={value.minAge}
          onChange={(v) => onChange({ ...value, maxAge: v })}
          testID="pref-max-age"
          value={value.maxAge}
        />
      </View>
      <View style={styles.field}>
        <Text variant="label" weight="semibold">
          Distance
        </Text>
        <View style={styles.wrap}>
          {DISTANCE_STEPS.map((d, index) => (
            <Chip
              key={String(d)}
              label={d === null ? "Anywhere" : `${d} km`}
              onPress={() => onChange({ ...value, maxDistanceKm: d })}
              selected={index === distanceIndex}
              testID={`pref-distance-${d ?? "any"}`}
            />
          ))}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  stack: { gap: spacing.xl },
  field: { gap: spacing.sm },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  stepper: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  stepperControls: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  stepButton: {
    width: 44,
    height: 44,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated,
    alignItems: "center",
    justifyContent: "center"
  },
  stepValue: { minWidth: 32, textAlign: "center" }
});
