import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { spacing } from "../theme";
import { Chip, Text } from "../ui";
import { genders, interestedInLabels } from "../utils/labels";
import { Stepper } from "./Stepper";

type Props = {
  value: Preferences;
  onChange: (v: Preferences) => void;
  error?: string | null;
};

export function PreferencesFields({ value, onChange, error }: Props) {
  const toggle = (g: Gender) => {
    const next = value.interestedIn.includes(g) ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g];
    onChange({ ...value, interestedIn: next });
  };
  return (
    <View style={styles.wrap}>
      <View style={styles.group}>
        <Text variant="label" color="textMuted">
          J’aimerais rencontrer
        </Text>
        <View style={styles.chips}>
          {genders.map((g) => (
            <Chip
              key={g}
              label={interestedInLabels[g]}
              selected={value.interestedIn.includes(g)}
              onPress={() => toggle(g)}
              testID={`interested-${g}`}
            />
          ))}
        </View>
        {error ? (
          <Text variant="caption" color="danger">
            {error}
          </Text>
        ) : null}
      </View>
      <View style={styles.group}>
        <Text variant="label" color="textMuted">
          Tranche d’âge
        </Text>
        <Stepper
          label="À partir de"
          value={value.minAge}
          min={18}
          max={value.maxAge}
          format={(v) => `${v} ans`}
          onChange={(minAge) => onChange({ ...value, minAge })}
        />
        <Stepper
          label="Jusqu'à"
          value={value.maxAge}
          min={value.minAge}
          max={99}
          format={(v) => `${v} ans`}
          onChange={(maxAge) => onChange({ ...value, maxAge })}
        />
      </View>
      <View style={styles.group}>
        <Text variant="label" color="textMuted">
          Distance maximale
        </Text>
        <Stepper
          label="Rayon"
          value={value.maxDistanceKm}
          min={5}
          max={300}
          step={5}
          format={(v) => `${v} km`}
          onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.lg
  },
  group: {
    gap: spacing.xs
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  }
});
