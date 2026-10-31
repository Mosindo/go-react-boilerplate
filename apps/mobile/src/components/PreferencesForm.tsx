import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { GENDER_LABELS } from "../lib/format";
import { Button, FormField, Text, spacing } from "../shared/ui";
import { Chip } from "./Chip";

type PreferencesFormProps = {
  value: Preferences;
  onChange: (next: Preferences) => void;
};

const DISTANCES = [10, 25, 50, 100, 250];

function Stepper({
  label,
  value,
  min,
  max,
  step = 1,
  onChange
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  onChange: (v: number) => void;
}) {
  return (
    <View style={styles.stepper}>
      <Text tone="secondary" weight="semibold">
        {label}
      </Text>
      <View style={styles.stepperControls}>
        <Button
          accessibilityLabel={`${label} moins`}
          disabled={value <= min}
          label="−"
          onPress={() => onChange(Math.max(min, value - step))}
          size="sm"
          variant="outline"
        />
        <Text style={styles.stepperValue} weight="bold">
          {value}
        </Text>
        <Button
          accessibilityLabel={`${label} plus`}
          disabled={value >= max}
          label="+"
          onPress={() => onChange(Math.min(max, value + step))}
          size="sm"
          variant="outline"
        />
      </View>
    </View>
  );
}

export function PreferencesForm({ value, onChange }: PreferencesFormProps) {
  const toggleGender = (g: Gender) => {
    const next = value.interestedIn.includes(g) ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g];
    onChange({ ...value, interestedIn: next });
  };

  return (
    <View style={styles.container}>
      <FormField label="Je souhaite rencontrer">
        <View style={styles.row}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip key={g} label={GENDER_LABELS[g]} onPress={() => toggleGender(g)} selected={value.interestedIn.includes(g)} />
          ))}
        </View>
      </FormField>
      <Stepper label="Âge minimum" max={value.ageMax} min={18} onChange={(v) => onChange({ ...value, ageMin: v })} value={value.ageMin} />
      <Stepper label="Âge maximum" max={99} min={value.ageMin} onChange={(v) => onChange({ ...value, ageMax: v })} value={value.ageMax} />
      <FormField label="Distance maximale">
        <View style={styles.row}>
          {DISTANCES.map((d) => (
            <Chip key={d} label={`${d} km`} onPress={() => onChange({ ...value, maxDistanceKm: d })} selected={value.maxDistanceKm === d} />
          ))}
        </View>
      </FormField>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { gap: spacing.lg },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  stepper: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  stepperControls: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  stepperValue: { minWidth: 32, textAlign: "center" }
});
