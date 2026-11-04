import React, { useState } from "react";
import { View } from "react-native";
import type { Gender, Preferences } from "../../api/types";
import { Button, Chip, Text, spacing } from "../../shared/ui";
import { GENDERS } from "../../utils/labels";
import { MAX_AGE, MIN_AGE, clamp, validatePreferences } from "../../utils/validation";
import { Stepper } from "./Stepper";

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 };
const DISTANCES = [10, 25, 50, 100, 250, 500];

type Props = { initial: Preferences | null; submitLabel: string; busy?: boolean; onSubmit: (p: Preferences) => void };

export function PreferencesForm({ initial, submitLabel, busy, onSubmit }: Props) {
  const [prefs, setPrefs] = useState<Preferences>(initial ?? DEFAULT_PREFERENCES);
  const [error, setError] = useState<string | null>(null);

  const toggle = (g: Gender) =>
    setPrefs((p) => ({ ...p, interestedIn: p.interestedIn.includes(g) ? p.interestedIn.filter((x) => x !== g) : [...p.interestedIn, g] }));

  const submit = () => {
    const problem = validatePreferences(prefs);
    setError(problem);
    if (!problem) onSubmit(prefs);
  };

  return (
    <View style={{ gap: spacing.lg }}>
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" tone="muted">Je souhaite rencontrer</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {GENDERS.map((g) => (
            <Chip key={g.value} label={g.plural} selected={prefs.interestedIn.includes(g.value)} onPress={() => toggle(g.value)} testID={`pref-${g.value}`} />
          ))}
        </View>
      </View>
      <Stepper label="Âge min." value={prefs.minAge} min={MIN_AGE} max={prefs.maxAge} unit=" ans" onChange={(v) => setPrefs((p) => ({ ...p, minAge: clamp(v, MIN_AGE, p.maxAge) }))} testID="pref-min-age" />
      <Stepper label="Âge max." value={prefs.maxAge} min={prefs.minAge} max={MAX_AGE} unit=" ans" onChange={(v) => setPrefs((p) => ({ ...p, maxAge: clamp(v, p.minAge, MAX_AGE) }))} testID="pref-max-age" />
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" tone="muted">Distance maximale</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {DISTANCES.map((d) => (
            <Chip key={d} label={`${d} km`} selected={prefs.maxDistanceKm === d} onPress={() => setPrefs((p) => ({ ...p, maxDistanceKm: d }))} />
          ))}
        </View>
        <Text variant="caption" tone="muted">La distance ne s'applique que si vous partagez votre position.</Text>
      </View>
      {error ? <Text tone="danger" accessibilityLiveRegion="polite">{error}</Text> : null}
      <Button title={submitLabel} onPress={submit} loading={busy} testID="prefs-submit" />
    </View>
  );
}
