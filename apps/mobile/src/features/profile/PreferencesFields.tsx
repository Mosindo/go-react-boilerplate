import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../../api/types";
import { GENDERS, GENDER_LABELS, MAX_AGE, MIN_AGE } from "../../lib/format";
import { Chip, FormField, Stepper, spacing } from "../../shared/ui";

export const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: [],
  minAge: 18,
  maxAge: 45,
  maxDistanceKm: 50
};

const DISTANCES: { label: string; value: number | null }[] = [
  { label: "10 km", value: 10 },
  { label: "25 km", value: 25 },
  { label: "50 km", value: 50 },
  { label: "100 km", value: 100 },
  { label: "300 km", value: 300 },
  { label: "Partout", value: null }
];

type PreferencesFieldsProps = {
  value: Preferences;
  onChange: (value: Preferences) => void;
  error?: string | null;
};

export function PreferencesFields({ value, onChange, error }: PreferencesFieldsProps) {
  const toggleGender = (gender: Gender) => {
    const next = value.interestedIn.includes(gender) ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender];
    onChange({ ...value, interestedIn: next });
  };

  return (
    <View style={styles.root}>
      <FormField error={error} label="Je veux rencontrer">
        <View style={styles.wrap}>
          {GENDERS.map((gender) => (
            <Chip
              key={gender}
              label={gender === "woman" ? "Des femmes" : gender === "man" ? "Des hommes" : "Des personnes non-binaires"}
              onPress={() => toggleGender(gender)}
              selected={value.interestedIn.includes(gender)}
              testID={`pref-${gender}`}
            />
          ))}
        </View>
      </FormField>

      <View style={styles.ages}>
        <FormField label="Âge minimum">
          <Stepper
            label="âge minimum"
            max={value.maxAge}
            min={MIN_AGE}
            onChange={(minAge) => onChange({ ...value, minAge })}
            value={value.minAge}
          />
        </FormField>
        <FormField label="Âge maximum">
          <Stepper
            label="âge maximum"
            max={MAX_AGE}
            min={value.minAge}
            onChange={(maxAge) => onChange({ ...value, maxAge })}
            value={value.maxAge}
          />
        </FormField>
      </View>

      <FormField hint="Seule une distance approximative est montrée aux autres." label="Distance maximale">
        <View style={styles.wrap}>
          {DISTANCES.map((distance) => (
            <Chip
              key={distance.label}
              label={distance.label}
              onPress={() => onChange({ ...value, maxDistanceKm: distance.value })}
              selected={value.maxDistanceKm === distance.value}
            />
          ))}
        </View>
      </FormField>
    </View>
  );
}

export { GENDER_LABELS };

const styles = StyleSheet.create({
  root: { gap: spacing.xl },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  ages: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xl }
});
