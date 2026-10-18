import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { Text, spacing } from "../shared/ui";
import { Chip, ChipRow } from "./Chip";
import { Stepper } from "./Stepper";
import { DISTANCE_OPTIONS, GENDERS, INTERESTED_IN_LABELS, MAX_AGE, MIN_AGE } from "./labels";

type PreferencesEditorProps = {
  value: Preferences;
  onChange: (next: Preferences) => void;
};

export function PreferencesEditor({ value, onChange }: PreferencesEditorProps) {
  const toggleGender = (gender: Gender) => {
    const has = value.interestedIn.includes(gender);
    if (has && value.interestedIn.length === 1) {
      return; // at least one gender must stay selected
    }
    onChange({
      ...value,
      interestedIn: has ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender]
    });
  };

  return (
    <View style={styles.wrap}>
      <View style={styles.block}>
        <Text tone="secondary" variant="label" weight="semibold">
          Je souhaite rencontrer
        </Text>
        <ChipRow>
          {GENDERS.map((gender) => (
            <Chip
              key={gender}
              label={INTERESTED_IN_LABELS[gender]}
              onPress={() => toggleGender(gender)}
              selected={value.interestedIn.includes(gender)}
              testID={`seek-${gender}`}
            />
          ))}
        </ChipRow>
      </View>

      <View style={styles.block}>
        <Stepper
          label="Âge minimum"
          max={value.ageMax}
          min={MIN_AGE}
          onChange={(ageMin) => onChange({ ...value, ageMin })}
          testID="age-min"
          value={value.ageMin}
        />
        <Stepper
          label="Âge maximum"
          max={MAX_AGE}
          min={value.ageMin}
          onChange={(ageMax) => onChange({ ...value, ageMax })}
          testID="age-max"
          value={value.ageMax}
        />
      </View>

      <View style={styles.block}>
        <Text tone="secondary" variant="label" weight="semibold">
          Distance maximale
        </Text>
        <ChipRow>
          {DISTANCE_OPTIONS.map((option) => (
            <Chip
              key={option.label}
              label={option.label}
              onPress={() => onChange({ ...value, maxDistanceKm: option.value })}
              selected={value.maxDistanceKm === option.value}
              testID={`distance-${option.value ?? "any"}`}
            />
          ))}
        </ChipRow>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xl },
  block: { gap: spacing.md }
});
