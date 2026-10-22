import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { MAX_AGE, MIN_AGE } from "../lib/validation";
import { Chip, FormField, Stepper, spacing } from "../shared/ui";

const OPTIONS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Des femmes" },
  { value: "man", label: "Des hommes" },
  { value: "non_binary", label: "Des personnes non-binaires" }
];

type PreferencesFormProps = { value: Preferences; onChange: (value: Preferences) => void };

export function PreferencesForm({ value, onChange }: PreferencesFormProps) {
  const toggle = (gender: Gender) => {
    const next = value.interestedIn.includes(gender) ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender];
    onChange({ ...value, interestedIn: next });
  };
  return (
    <View style={styles.root}>
      <FormField label="Je souhaite rencontrer">
        <View style={styles.chips}>
          {OPTIONS.map((option) => (
            <Chip
              key={option.value}
              label={option.label}
              onPress={() => toggle(option.value)}
              selected={value.interestedIn.includes(option.value)}
              testID={`interested-${option.value}`}
            />
          ))}
        </View>
      </FormField>
      <Stepper
        format={(v) => `${v} ans`}
        label="Âge minimum"
        max={Math.min(MAX_AGE, value.maxAge)}
        min={MIN_AGE}
        onChange={(minAge) => onChange({ ...value, minAge })}
        testID="min-age"
        value={value.minAge}
      />
      <Stepper
        format={(v) => `${v} ans`}
        label="Âge maximum"
        max={MAX_AGE}
        min={Math.max(MIN_AGE, value.minAge)}
        onChange={(maxAge) => onChange({ ...value, maxAge })}
        testID="max-age"
        value={value.maxAge}
      />
      <Stepper
        format={(v) => `${v} km`}
        label="Distance maximale"
        max={500}
        min={5}
        onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
        step={5}
        testID="max-distance"
        value={value.maxDistanceKm}
      />
    </View>
  );
}

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 };

const styles = StyleSheet.create({
  root: { gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
