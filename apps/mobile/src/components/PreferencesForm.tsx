import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { GENDER_LABELS } from "../api/dating";
import type { Gender, Preferences } from "../api/types";
import { MAX_AGE, MIN_AGE } from "../lib/validation";
import { radii, spacing, useTheme } from "../theme";
import { Chip, Text } from "../ui";

export const DISTANCE_OPTIONS: { label: string; km: number }[] = [
  { label: "10 km", km: 10 },
  { label: "25 km", km: 25 },
  { label: "50 km", km: 50 },
  { label: "100 km", km: 100 },
  { label: "250 km", km: 250 },
  { label: "Partout", km: 20000 }
];

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: [], ageMin: 18, ageMax: 50, maxDistanceKm: 50 };

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
  const button = (delta: number, icon: "remove" | "add", enabled: boolean) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${delta < 0 ? "Diminuer" : "Augmenter"} ${label}`}
      disabled={!enabled}
      onPress={() => onChange(value + delta)}
      style={[styles.stepButton, { borderColor: colors.border, backgroundColor: colors.surface, opacity: enabled ? 1 : 0.4 }]}
    >
      <Ionicons name={icon} size={20} color={colors.text} />
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <View style={styles.stepRow}>
        {button(-1, "remove", value > min)}
        <Text variant="heading" style={styles.stepValue} accessibilityLiveRegion="polite">
          {value}
        </Text>
        {button(1, "add", value < max)}
      </View>
    </View>
  );
}

type Props = { value: Preferences; onChange: (next: Preferences) => void; error?: string | null };

export function PreferencesFields({ value, onChange, error }: Props) {
  const toggleGender = (g: Gender) => {
    const has = value.interestedIn.includes(g);
    onChange({ ...value, interestedIn: has ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g] });
  };
  return (
    <View style={styles.group}>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Je souhaite rencontrer
        </Text>
        <View style={styles.chips}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip key={g} label={GENDER_LABELS[g]} selected={value.interestedIn.includes(g)} onPress={() => toggleGender(g)} />
          ))}
        </View>
        {error ? (
          <Text variant="caption" tone="danger">
            {error}
          </Text>
        ) : null}
      </View>
      <View style={styles.ages}>
        <Stepper
          label="Âge min."
          value={value.ageMin}
          min={MIN_AGE}
          max={value.ageMax}
          onChange={(ageMin) => onChange({ ...value, ageMin })}
        />
        <Stepper
          label="Âge max."
          value={value.ageMax}
          min={value.ageMin}
          max={MAX_AGE}
          onChange={(ageMax) => onChange({ ...value, ageMax })}
        />
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Distance maximale
        </Text>
        <View style={styles.chips}>
          {DISTANCE_OPTIONS.map((o) => (
            <Chip
              key={o.km}
              label={o.label}
              selected={value.maxDistanceKm === o.km}
              onPress={() => onChange({ ...value, maxDistanceKm: o.km })}
            />
          ))}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  group: { gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  ages: { flexDirection: "row", gap: spacing.xl },
  stepper: { gap: spacing.sm },
  stepRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  stepButton: { width: 40, height: 40, borderRadius: radii.pill, borderWidth: 1.5, alignItems: "center", justifyContent: "center" },
  stepValue: { minWidth: 36, textAlign: "center" }
});
