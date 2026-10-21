import React from "react";
import { StyleSheet, View } from "react-native";
import type { Preferences } from "../api/types";
import { distanceLabel, INTERESTED_IN } from "../lib/labels";
import { stepDistance, toggleGender } from "../lib/preferences";
import { clampAgeRange } from "../lib/validation";
import { Chip, FormField, Stepper } from "../shared/ui";
import { spacing } from "../theme";

type Props = { value: Preferences; onChange: (next: Preferences) => void };

/** Who I want to meet: genders, age range, distance. */
export function PreferencesFields({ value, onChange }: Props) {
  const setAge = (min: number, max: number, changed: "min" | "max") => {
    const [minAge, maxAge] = clampAgeRange(min, max, changed);
    onChange({ ...value, minAge, maxAge });
  };
  return (
    <View style={styles.form}>
      <FormField label="Je souhaite rencontrer">
        <View style={styles.chips}>
          {INTERESTED_IN.map((g) => (
            <Chip
              key={g.value}
              testID={`interested-${g.value}`}
              label={g.label}
              selected={value.interestedIn.includes(g.value)}
              onPress={() => onChange({ ...value, interestedIn: toggleGender(value.interestedIn, g.value) })}
            />
          ))}
        </View>
      </FormField>
      <Stepper
        testID="pref-min-age"
        label="Âge minimum"
        value={`${value.minAge} ans`}
        onDecrement={() => setAge(value.minAge - 1, value.maxAge, "min")}
        onIncrement={() => setAge(value.minAge + 1, value.maxAge, "min")}
      />
      <Stepper
        testID="pref-max-age"
        label="Âge maximum"
        value={`${value.maxAge} ans`}
        onDecrement={() => setAge(value.minAge, value.maxAge - 1, "max")}
        onIncrement={() => setAge(value.minAge, value.maxAge + 1, "max")}
      />
      <Stepper
        testID="pref-distance"
        label="Distance maximale"
        value={distanceLabel(value.maxDistanceKm)}
        onDecrement={() => onChange({ ...value, maxDistanceKm: stepDistance(value.maxDistanceKm, -1) })}
        onIncrement={() => onChange({ ...value, maxDistanceKm: stepDistance(value.maxDistanceKm, 1) })}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
