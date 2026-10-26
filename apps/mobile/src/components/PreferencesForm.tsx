import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api";
import { interestedLabel } from "../lib/format";
import { radius, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Chip } from "./Chip";
import { Text } from "./Text";

const GENDERS: Gender[] = ["woman", "man", "nonbinary"];
const DISTANCES = [10, 25, 50, 100, 250, 1000];

type Props = {
  initial?: Preferences | null;
  submitLabel: string;
  loading?: boolean;
  onSubmit: (prefs: Preferences) => void;
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
  const theme = useTheme();
  const btn = (glyph: string, delta: number, name: string) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={name}
      onPress={() => onChange(Math.min(max, Math.max(min, value + delta)))}
      style={[styles.stepBtn, { borderColor: theme.border, backgroundColor: theme.surface }]}
    >
      <Text variant="heading">{glyph}</Text>
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text variant="label" muted>
        {label}
      </Text>
      <View style={styles.stepRow}>
        {btn("−", -1, `Diminuer ${label}`)}
        <Text variant="heading" style={styles.stepValue} accessibilityLabel={`${label} : ${value}`}>
          {value}
        </Text>
        {btn("+", 1, `Augmenter ${label}`)}
      </View>
    </View>
  );
}

export function PreferencesForm({ initial, submitLabel, loading, onSubmit }: Props) {
  const [interestedIn, setInterestedIn] = useState<Gender[]>(initial?.interestedIn ?? []);
  const [minAge, setMinAge] = useState(initial?.minAge ?? 18);
  const [maxAge, setMaxAge] = useState(initial?.maxAge ?? 45);
  const [distance, setDistance] = useState(initial?.maxDistanceKm ?? 50);

  const toggle = (g: Gender) =>
    setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]));
  const valid = interestedIn.length > 0 && minAge <= maxAge;

  return (
    <View style={styles.form}>
      <View style={styles.section}>
        <Text variant="heading">Je souhaite rencontrer</Text>
        <View style={styles.row}>
          {GENDERS.map((g) => (
            <Chip
              key={g}
              label={interestedLabel(g)}
              selected={interestedIn.includes(g)}
              onPress={() => toggle(g)}
              testID={`interested-${g}`}
            />
          ))}
        </View>
      </View>
      <View style={styles.section}>
        <Text variant="heading">Tranche d&apos;âge</Text>
        <View style={styles.row}>
          <Stepper label="Âge min." value={minAge} min={18} max={maxAge} onChange={setMinAge} />
          <Stepper label="Âge max." value={maxAge} min={minAge} max={99} onChange={setMaxAge} />
        </View>
      </View>
      <View style={styles.section}>
        <Text variant="heading">Distance maximale</Text>
        <View style={styles.row}>
          {DISTANCES.map((d) => (
            <Chip
              key={d}
              label={d >= 1000 ? "Partout" : `${d} km`}
              selected={distance === d}
              onPress={() => setDistance(d)}
            />
          ))}
        </View>
      </View>
      <Button
        label={submitLabel}
        loading={loading}
        disabled={!valid}
        onPress={() => onSubmit({ interestedIn, minAge, maxAge, maxDistanceKm: distance })}
        testID="save-preferences"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.xl },
  section: { gap: spacing.md },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm, alignItems: "flex-end" },
  stepper: { gap: spacing.xs, marginRight: spacing.xl },
  stepRow: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  stepBtn: {
    width: 44,
    height: 44,
    borderRadius: radius.pill,
    borderWidth: 1.5,
    alignItems: "center",
    justifyContent: "center"
  },
  stepValue: { minWidth: 36, textAlign: "center" }
});
