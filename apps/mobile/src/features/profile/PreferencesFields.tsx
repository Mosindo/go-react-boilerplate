import React from "react";
import { StyleSheet, View } from "react-native";
import { GENDERS, type Gender, type Preferences } from "../../api/types";
import { clampAgeRange, clampDistance } from "../../lib/validation";
import { spacing } from "../../theme";
import { Chip, Stepper, Text } from "../../ui";
import { GENDER_LABELS, INTERESTED_IN_LABELS } from "./labels";

const DISTANCE_PRESETS = [10, 25, 50, 100, 250, 500];

type Props = { value: Preferences; onChange: (next: Preferences) => void };

/** Who you're interested in, age range and max distance. Shared by onboarding and settings. */
export function PreferencesFields({ value, onChange }: Props) {
  const toggleGender = (gender: Gender) => {
    const has = value.interestedIn.includes(gender);
    const interestedIn = has ? value.interestedIn.filter((g) => g !== gender) : [...value.interestedIn, gender];
    onChange({ ...value, interestedIn });
  };
  const presets = DISTANCE_PRESETS.includes(value.maxDistanceKm)
    ? DISTANCE_PRESETS
    : [...DISTANCE_PRESETS, value.maxDistanceKm].sort((a, b) => a - b);

  return (
    <View style={styles.wrap}>
      <Text weight="semibold">Show me</Text>
      <View style={styles.chips}>
        {GENDERS.map((gender) => (
          <Chip
            key={gender}
            label={INTERESTED_IN_LABELS[gender]}
            onPress={() => toggleGender(gender)}
            selected={value.interestedIn.includes(gender)}
          />
        ))}
      </View>
      {value.interestedIn.length === 0 ? (
        <Text tone="danger" variant="small">
          Choose at least one option.
        </Text>
      ) : null}

      <Text style={styles.gap} weight="semibold">
        Age range
      </Text>
      <Stepper
        label="Minimum age"
        max={99}
        min={18}
        onChange={(minAge) => onChange({ ...value, ...clampAgeRange({ minAge, maxAge: value.maxAge }, "min") })}
        value={value.minAge}
      />
      <Stepper
        label="Maximum age"
        max={99}
        min={18}
        onChange={(maxAge) => onChange({ ...value, ...clampAgeRange({ minAge: value.minAge, maxAge }, "max") })}
        value={value.maxAge}
      />

      <Text style={styles.gap} weight="semibold">
        Maximum distance
      </Text>
      <View style={styles.chips}>
        {presets.map((km) => (
          <Chip
            key={km}
            label={`${km} km`}
            onPress={() => onChange({ ...value, maxDistanceKm: clampDistance(km) })}
            selected={value.maxDistanceKm === km}
          />
        ))}
      </View>
    </View>
  );
}

export function genderLabel(gender: Gender): string {
  return GENDER_LABELS[gender];
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", columnGap: spacing.xs },
  gap: { marginTop: spacing.sm }
});
