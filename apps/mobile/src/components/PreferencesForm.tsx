import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Gender, Preferences } from "../api/types";
import { MAX_AGE, MIN_AGE } from "../lib/validation";
import { clamp } from "../lib/format";
import { Chip, Text, radii, spacing, useTheme } from "../ui";

const GENDER_OPTIONS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Des femmes" },
  { value: "man", label: "Des hommes" },
  { value: "non_binary", label: "Des personnes non-binaires" }
];
const DISTANCES = [10, 25, 50, 100, 250];

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 };

export function validatePreferences(prefs: Preferences): string | null {
  if (prefs.interestedIn.length === 0) return "Choisissez au moins une option.";
  if (prefs.minAge > prefs.maxAge) return "L'âge minimum doit être inférieur à l'âge maximum.";
  return null;
}

function Stepper({
  label,
  value,
  min,
  max,
  onChange
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
}) {
  const { colors } = useTheme();
  const button = (icon: "remove" | "add", delta: number) => (
    <Pressable
      accessibilityLabel={`${delta < 0 ? "Diminuer" : "Augmenter"} ${label}`}
      accessibilityRole="button"
      onPress={() => onChange(clamp(value + delta, min, max))}
      style={[styles.stepButton, { borderColor: colors.border, backgroundColor: colors.surface }]}
    >
      <Ionicons color={colors.text} name={icon} size={20} />
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <View style={styles.stepperRow}>
        {button("remove", -1)}
        <Text variant="heading" style={styles.stepValue} testID={`pref-${label}`}>
          {value}
        </Text>
        {button("add", 1)}
      </View>
    </View>
  );
}

type Props = { value: Preferences; onChange: (next: Preferences) => void };

export function PreferencesForm({ value, onChange }: Props) {
  const toggle = (gender: Gender) => {
    const has = value.interestedIn.includes(gender);
    onChange({
      ...value,
      interestedIn: has ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender]
    });
  };

  return (
    <View style={styles.wrap}>
      <View style={styles.group}>
        <Text variant="heading">Je souhaite rencontrer</Text>
        <View style={styles.chips}>
          {GENDER_OPTIONS.map((option) => (
            <Chip
              key={option.value}
              label={option.label}
              onPress={() => toggle(option.value)}
              selected={value.interestedIn.includes(option.value)}
              testID={`interested-${option.value}`}
            />
          ))}
        </View>
      </View>

      <View style={styles.group}>
        <Text variant="heading">Tranche d'âge</Text>
        <View style={styles.row}>
          <Stepper
            label="Min"
            max={value.maxAge}
            min={MIN_AGE}
            onChange={(minAge) => onChange({ ...value, minAge })}
            value={value.minAge}
          />
          <Stepper
            label="Max"
            max={MAX_AGE}
            min={value.minAge}
            onChange={(maxAge) => onChange({ ...value, maxAge })}
            value={value.maxAge}
          />
        </View>
      </View>

      <View style={styles.group}>
        <Text variant="heading">Distance maximale</Text>
        <View style={styles.chips}>
          {DISTANCES.map((km) => (
            <Chip
              key={km}
              label={`${km} km`}
              onPress={() => onChange({ ...value, maxDistanceKm: km })}
              selected={value.maxDistanceKm === km}
            />
          ))}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xl },
  group: { gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  row: { flexDirection: "row", gap: spacing.xl },
  stepper: { gap: spacing.xs },
  stepperRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  stepButton: {
    width: 44,
    height: 44,
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  stepValue: { minWidth: 34, textAlign: "center" }
});
