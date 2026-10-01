import React from "react";
import { StyleSheet, View } from "react-native";
import { GENDERS, type Gender, type Preferences } from "../api/types";
import { Chip, FormField, IconButton, Text, spacing } from "../shared/ui";
import { MAX_AGE, MIN_AGE } from "../utils/validation";

const DISTANCES = [
  { km: 10, label: "10 km" },
  { km: 25, label: "25 km" },
  { km: 50, label: "50 km" },
  { km: 100, label: "100 km" },
  { km: 250, label: "250 km" },
  { km: 20000, label: "Partout" }
] as const;

type PreferencesEditorProps = {
  value: Preferences;
  onChange: (next: Preferences) => void;
};

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
  return (
    <View accessibilityLabel={`${label} : ${value}`} accessibilityRole="adjustable" style={styles.stepper}>
      <IconButton disabled={value <= min} icon="remove" label={`Diminuer ${label}`} onPress={() => onChange(value - 1)} size={40} />
      <View style={styles.stepperValue}>
        <Text variant="heading">{value}</Text>
        <Text tone="muted" variant="caption">
          {label}
        </Text>
      </View>
      <IconButton disabled={value >= max} icon="add" label={`Augmenter ${label}`} onPress={() => onChange(value + 1)} size={40} />
    </View>
  );
}

/** Who I want to meet: genders, age range and distance. */
export function PreferencesEditor({ value, onChange }: PreferencesEditorProps) {
  const toggleGender = (gender: Gender) => {
    const next = value.interestedIn.includes(gender) ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender];
    if (next.length > 0) {
      onChange({ ...value, interestedIn: next });
    }
  };

  return (
    <View style={styles.root}>
      <FormField hint="Vous pouvez en choisir plusieurs." label="Je souhaite rencontrer">
        <View style={styles.row}>
          {GENDERS.map((g) => (
            <Chip key={g.value} label={g.label} onPress={() => toggleGender(g.value)} selected={value.interestedIn.includes(g.value)} />
          ))}
        </View>
      </FormField>

      <FormField label="Tranche d'âge">
        <View style={styles.row}>
          <Stepper
            label="âge min"
            max={value.ageMax}
            min={MIN_AGE}
            onChange={(ageMin) => onChange({ ...value, ageMin })}
            value={value.ageMin}
          />
          <Stepper
            label="âge max"
            max={MAX_AGE}
            min={value.ageMin}
            onChange={(ageMax) => onChange({ ...value, ageMax })}
            value={value.ageMax}
          />
        </View>
      </FormField>

      <FormField hint="Seule une distance approximative est affichée aux autres." label="Distance maximale">
        <View style={styles.row}>
          {DISTANCES.map((d) => (
            <Chip
              key={d.km}
              label={d.label}
              onPress={() => onChange({ ...value, maxDistanceKm: d.km })}
              selected={value.maxDistanceKm === d.km}
            />
          ))}
        </View>
      </FormField>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.lg },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm, alignItems: "center" },
  stepper: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  stepperValue: { minWidth: 56, alignItems: "center" }
});
