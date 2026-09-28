import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { INTERESTED_IN_LABEL } from "../domain/labels";
import { spacing } from "../theme";
import { Chip, FormField, Text } from "../shared/ui";
import { GenderChips } from "./GenderChips";
import { Stepper } from "./Stepper";

const DISTANCE_PRESETS = [10, 25, 50, 100, 250, 500];

export type PreferencesFormProps = {
  value: Preferences;
  onChange: (next: Preferences) => void;
  error?: string | null;
};

export function PreferencesForm({ value, onChange, error }: PreferencesFormProps) {
  const toggleGender = (gender: Gender) => {
    const next = value.interestedIn.includes(gender)
      ? value.interestedIn.filter((item) => item !== gender)
      : [...value.interestedIn, gender];
    onChange({ ...value, interestedIn: next });
  };
  const distances = DISTANCE_PRESETS.includes(value.maxDistanceKm)
    ? DISTANCE_PRESETS
    : [...DISTANCE_PRESETS, value.maxDistanceKm].sort((a, b) => a - b);

  return (
    <View style={styles.wrap}>
      <FormField error={error} label="Show me">
        <GenderChips labels={INTERESTED_IN_LABEL} onToggle={toggleGender} selected={value.interestedIn} />
      </FormField>
      <FormField label="Age range">
        <Stepper
          label="Minimum age"
          max={value.maxAge}
          min={18}
          onChange={(minAge) => onChange({ ...value, minAge })}
          value={value.minAge}
        />
        <Stepper
          label="Maximum age"
          max={99}
          min={value.minAge}
          onChange={(maxAge) => onChange({ ...value, maxAge })}
          value={value.maxAge}
        />
      </FormField>
      <FormField label="Maximum distance">
        <View style={styles.chips}>
          {distances.map((distance) => (
            <Chip
              key={distance}
              label={`${distance} km`}
              onPress={() => onChange({ ...value, maxDistanceKm: distance })}
              selected={value.maxDistanceKm === distance}
            />
          ))}
        </View>
        <Text tone="muted" variant="caption">
          Only people who also fit your profile will see you.
        </Text>
      </FormField>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xl },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
