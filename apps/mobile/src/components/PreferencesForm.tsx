import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { GENDER_LABELS } from "../lib/format";
import { Button } from "../ui/Button";
import { Chip } from "../ui/Chip";
import { Text } from "../ui/Text";

type Props = {
  initial: Preferences;
  saving: boolean;
  submitLabel: string;
  onSubmit: (p: Preferences) => void;
};

const AGE_PRESETS: [number, number][] = [
  [18, 30],
  [25, 40],
  [35, 55],
  [18, 99]
];
const DISTANCES = [10, 25, 50, 100, 500];

export function PreferencesForm({ initial, saving, submitLabel, onSubmit }: Props) {
  const [interestedIn, setInterestedIn] = useState<Gender[]>(initial.interestedIn);
  const [ageMin, setAgeMin] = useState(initial.ageMin);
  const [ageMax, setAgeMax] = useState(initial.ageMax);
  const [distance, setDistance] = useState(initial.maxDistanceKm);

  const toggle = (g: Gender) =>
    setInterestedIn((current) =>
      current.includes(g) ? current.filter((x) => x !== g) : [...current, g]
    );

  return (
    <View style={styles.form}>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          I would like to meet
        </Text>
        <View style={styles.row}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip
              key={g}
              label={GENDER_LABELS[g]}
              selected={interestedIn.includes(g)}
              onPress={() => toggle(g)}
            />
          ))}
        </View>
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Age range: {ageMin}–{ageMax}
        </Text>
        <View style={[styles.row, styles.wrap]}>
          {AGE_PRESETS.map(([min, max]) => (
            <Chip
              key={`${min}-${max}`}
              label={`${min}–${max}`}
              selected={ageMin === min && ageMax === max}
              onPress={() => {
                setAgeMin(min);
                setAgeMax(max);
              }}
            />
          ))}
        </View>
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Maximum distance
        </Text>
        <View style={[styles.row, styles.wrap]}>
          {DISTANCES.map((d) => (
            <Chip
              key={d}
              label={`${d} km`}
              selected={distance === d}
              onPress={() => setDistance(d)}
            />
          ))}
        </View>
      </View>
      <Button
        label={submitLabel}
        loading={saving}
        disabled={interestedIn.length === 0}
        onPress={() => onSubmit({ interestedIn, ageMin, ageMax, maxDistanceKm: distance })}
        testID="preferences-submit"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: 20 },
  group: { gap: 8 },
  row: { flexDirection: "row", gap: 8 },
  wrap: { flexWrap: "wrap" }
});
