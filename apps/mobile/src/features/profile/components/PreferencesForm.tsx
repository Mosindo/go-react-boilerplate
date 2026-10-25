import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../../../api";
import { DEFAULT_DISTANCE_KM, MAX_AGE, MAX_DISTANCE_KM, MIN_AGE } from "../../../config";
import { Chip, Stepper, SwitchRow, Text } from "../../../shared/ui";
import { GENDERS, INTERESTED_IN_LABELS } from "../../../shared/labels";
import { spacing } from "../../../theme";

type Props = { value: Preferences; onChange: (value: Preferences) => void };

export const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: [],
  minAge: MIN_AGE,
  maxAge: 40,
  maxDistanceKm: DEFAULT_DISTANCE_KM
};

export function PreferencesForm({ value, onChange }: Props) {
  const toggleGender = (gender: Gender) => {
    const interestedIn = value.interestedIn.includes(gender)
      ? value.interestedIn.filter((item) => item !== gender)
      : [...value.interestedIn, gender];
    onChange({ ...value, interestedIn });
  };

  return (
    <View style={styles.wrapper}>
      <View style={styles.group}>
        <Text color="textMuted" variant="caption">
          Je souhaite rencontrer
        </Text>
        <View style={styles.chips}>
          {GENDERS.map((gender) => (
            <Chip
              key={gender}
              label={INTERESTED_IN_LABELS[gender]}
              onPress={() => toggleGender(gender)}
              selected={value.interestedIn.includes(gender)}
              testID={`interested-${gender}`}
            />
          ))}
        </View>
      </View>
      <Stepper
        format={(age) => `${age} ans`}
        label="Âge minimum"
        max={value.maxAge}
        min={MIN_AGE}
        onChange={(minAge) => onChange({ ...value, minAge })}
        value={value.minAge}
      />
      <Stepper
        format={(age) => `${age} ans`}
        label="Âge maximum"
        max={MAX_AGE}
        min={value.minAge}
        onChange={(maxAge) => onChange({ ...value, maxAge })}
        value={value.maxAge}
      />
      <SwitchRow
        label="Limiter la distance"
        onValueChange={(limited) => onChange({ ...value, maxDistanceKm: limited ? DEFAULT_DISTANCE_KM : null })}
        value={value.maxDistanceKm !== null}
      />
      {value.maxDistanceKm !== null ? (
        <Stepper
          format={(km) => `${km} km`}
          label="Distance maximale"
          max={MAX_DISTANCE_KM}
          min={5}
          onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
          step={5}
          value={value.maxDistanceKm}
        />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: { gap: spacing.lg },
  group: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
