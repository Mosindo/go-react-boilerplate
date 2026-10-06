import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Gender, Preferences } from "../../api/types";
import { genderLabel } from "../../lib/format";
import { MAX_AGE, MIN_AGE, clampAgeRange } from "../../lib/validation";
import { Badge, Button, FormField, Notice, Text, spacing, useTheme } from "../../shared/ui";

const GENDERS: Gender[] = ["woman", "man", "non_binary"];
const DISTANCES = [10, 25, 50, 100, 200];

type PreferencesFormProps = {
  initial?: Preferences | null;
  submitLabel: string;
  busy?: boolean;
  error?: string | null;
  onSubmit: (values: Preferences) => void;
};

function Stepper({
  label,
  value,
  onChange,
  min,
  max
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
  min: number;
  max: number;
}) {
  const { colors } = useTheme();
  const button = (icon: "remove" | "add", delta: number, disabled: boolean) => (
    <Pressable
      accessibilityLabel={`${delta < 0 ? "Diminuer" : "Augmenter"} ${label}`}
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      onPress={() => onChange(value + delta)}
      style={[styles.stepButton, { backgroundColor: colors.primarySoft, opacity: disabled ? 0.4 : 1 }]}
    >
      <Ionicons color={colors.primary} name={icon} size={20} />
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text tone="muted" variant="label">
        {label}
      </Text>
      <View style={styles.stepperRow}>
        {button("remove", -1, value <= min)}
        <Text accessibilityLabel={`${label} : ${value} ans`} style={styles.stepValue} variant="heading">
          {value}
        </Text>
        {button("add", 1, value >= max)}
      </View>
    </View>
  );
}

export function PreferencesForm({ initial, submitLabel, busy = false, error, onSubmit }: PreferencesFormProps) {
  const [interestedIn, setInterestedIn] = useState<Gender[]>(initial?.interestedIn ?? []);
  const [minAge, setMinAge] = useState(initial?.minAge ?? 18);
  const [maxAge, setMaxAge] = useState(initial?.maxAge ?? 45);
  const [distance, setDistance] = useState(initial?.maxDistanceKm ?? 50);
  const [problem, setProblem] = useState<string | null>(null);

  const toggle = (gender: Gender) => {
    setInterestedIn((current) => (current.includes(gender) ? current.filter((g) => g !== gender) : [...current, gender]));
  };

  const submit = () => {
    if (interestedIn.length === 0) {
      setProblem("Choisissez au moins une option.");
      return;
    }
    setProblem(null);
    const range = clampAgeRange(minAge, maxAge);
    onSubmit({ interestedIn, minAge: range.min, maxAge: range.max, maxDistanceKm: distance });
  };

  return (
    <View style={styles.root}>
      <FormField error={problem} label="Je souhaite rencontrer">
        <View style={styles.chips}>
          {GENDERS.map((option) => (
            <Badge key={option} label={genderLabel(option)} onPress={() => toggle(option)} selected={interestedIn.includes(option)} />
          ))}
        </View>
      </FormField>

      <FormField helperText="Vous ne verrez que des profils dont la tranche d'âge inclut aussi la vôtre." label="Tranche d'âge">
        <View style={styles.ages}>
          <Stepper label="Min" max={maxAge} min={MIN_AGE} onChange={setMinAge} value={minAge} />
          <Stepper label="Max" max={MAX_AGE} min={minAge} onChange={setMaxAge} value={maxAge} />
        </View>
      </FormField>

      <FormField label="Distance maximale">
        <View style={styles.chips}>
          {DISTANCES.map((km) => (
            <Badge key={km} label={`${km} km`} onPress={() => setDistance(km)} selected={distance === km} />
          ))}
        </View>
      </FormField>

      {error ? <Notice message={error} tone="danger" /> : null}
      <Button label={submitLabel} loading={busy} onPress={submit} testID="preferences-submit" />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.xl },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  ages: { flexDirection: "row", gap: spacing.xl },
  stepper: { gap: spacing.xs },
  stepperRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  stepButton: { width: 36, height: 36, borderRadius: 18, alignItems: "center", justifyContent: "center" },
  stepValue: { minWidth: 36, textAlign: "center" }
});
