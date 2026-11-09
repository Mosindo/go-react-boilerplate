import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { radii, useTheme } from "../theme/theme";
import { Button, Chip, Text } from "../ui";

const OPTIONS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Des femmes" },
  { value: "man", label: "Des hommes" },
  { value: "nonbinary", label: "Des personnes non-binaires" }
];

const DISTANCES = [10, 25, 50, 100, 250, 500];

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 };

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
  const { colors } = useTheme();
  const btn = (text: string, delta: number, enabled: boolean) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${text === "−" ? "Diminuer" : "Augmenter"} ${label}`}
      disabled={!enabled}
      onPress={() => onChange(value + delta)}
      style={[styles.step, { backgroundColor: colors.surfaceAlt, opacity: enabled ? 1 : 0.4 }]}
    >
      <Text variant="heading">{text}</Text>
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text variant="label" muted>
        {label}
      </Text>
      <View style={styles.stepControls}>
        {btn("−", -1, value > min)}
        <Text variant="heading" style={styles.stepValue}>
          {value}
        </Text>
        {btn("+", 1, value < max)}
      </View>
    </View>
  );
}

type Props = { initial: Preferences | null; submitLabel: string; busy: boolean; onSubmit: (p: Preferences) => void };

export function PreferencesForm({ initial, submitLabel, busy, onSubmit }: Props) {
  const [prefs, setPrefs] = useState<Preferences>(initial ?? DEFAULT_PREFERENCES);
  const [touched, setTouched] = useState(false);
  const toggle = (g: Gender) =>
    setPrefs((p) => ({ ...p, interestedIn: p.interestedIn.includes(g) ? p.interestedIn.filter((x) => x !== g) : [...p.interestedIn, g] }));

  const submit = () => {
    setTouched(true);
    if (prefs.interestedIn.length > 0) onSubmit(prefs);
  };

  return (
    <View style={styles.form}>
      <View style={styles.group}>
        <Text variant="label" muted>
          Je souhaite rencontrer
        </Text>
        <View style={styles.wrap}>
          {OPTIONS.map((o) => (
            <Chip key={o.value} label={o.label} selected={prefs.interestedIn.includes(o.value)} onPress={() => toggle(o.value)} />
          ))}
        </View>
        {touched && prefs.interestedIn.length === 0 ? (
          <Text variant="caption" color="#B3261E">
            Choisissez au moins une option.
          </Text>
        ) : null}
      </View>
      <Stepper
        label="Âge minimum"
        value={prefs.minAge}
        min={18}
        max={prefs.maxAge}
        onChange={(v) => setPrefs((p) => ({ ...p, minAge: v }))}
      />
      <Stepper
        label="Âge maximum"
        value={prefs.maxAge}
        min={prefs.minAge}
        max={100}
        onChange={(v) => setPrefs((p) => ({ ...p, maxAge: v }))}
      />
      <View style={styles.group}>
        <Text variant="label" muted>
          Distance maximale
        </Text>
        <View style={styles.wrap}>
          {DISTANCES.map((d) => (
            <Chip
              key={d}
              label={`${d} km`}
              selected={prefs.maxDistanceKm === d}
              onPress={() => setPrefs((p) => ({ ...p, maxDistanceKm: d }))}
            />
          ))}
        </View>
        <Text variant="caption" muted>
          La distance ne s'applique que si vous et l'autre personne avez partagé une position.
        </Text>
      </View>
      <Button title={submitLabel} onPress={submit} loading={busy} testID="prefs-submit" />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: 20 },
  group: { gap: 8 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  stepper: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  stepControls: { flexDirection: "row", alignItems: "center", gap: 12 },
  step: { width: 44, height: 44, borderRadius: radii.pill, alignItems: "center", justifyContent: "center" },
  stepValue: { minWidth: 36, textAlign: "center" }
});
