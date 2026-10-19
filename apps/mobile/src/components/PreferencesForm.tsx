import React, { useState } from "react";
import { StyleSheet, Switch, View } from "react-native";
import { Button, Chip, Text, useToast } from "../ui";
import { spacing } from "../theme/tokens";
import { errorMessage } from "../lib/errors";
import { GENDERS } from "./ProfileForm";
import { Stepper } from "./Stepper";
import type { Gender, Preferences } from "../api/types";

const DISTANCES: { label: string; value: number | null }[] = [
  { label: "10 km", value: 10 },
  { label: "25 km", value: 25 },
  { label: "50 km", value: 50 },
  { label: "100 km", value: 100 },
  { label: "Partout", value: null }
];

export const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: [],
  minAge: 18,
  maxAge: 60,
  maxDistanceKm: 50,
  showDistance: true
};

type Props = {
  initial?: Preferences | null;
  submitLabel: string;
  onSubmit: (prefs: Preferences) => Promise<void>;
};

export function PreferencesForm({ initial, submitLabel, onSubmit }: Props) {
  const toast = useToast();
  const [prefs, setPrefs] = useState<Preferences>(initial ?? DEFAULT_PREFERENCES);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toggleGender = (gender: Gender) =>
    setPrefs((p) => ({
      ...p,
      interestedIn: p.interestedIn.includes(gender)
        ? p.interestedIn.filter((g) => g !== gender)
        : [...p.interestedIn, gender]
    }));

  const submit = async () => {
    if (prefs.interestedIn.length === 0) {
      setError("Sélectionnez au moins une option.");
      return;
    }
    setError(null);
    setSaving(true);
    try {
      await onSubmit(prefs);
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <View style={styles.form}>
      <View style={styles.group}>
        <Text variant="caption" tone="muted" bold>
          Je souhaite rencontrer
        </Text>
        <View style={styles.row}>
          {GENDERS.map((g) => (
            <Chip
              key={g.value}
              label={g.label}
              selected={prefs.interestedIn.includes(g.value)}
              onPress={() => toggleGender(g.value)}
              testID={`interested-${g.value}`}
            />
          ))}
        </View>
        {error ? (
          <Text variant="caption" tone="danger">
            {error}
          </Text>
        ) : null}
      </View>

      <View style={styles.group}>
        <Text variant="caption" tone="muted" bold>
          Tranche d&apos;âge : {prefs.minAge} – {prefs.maxAge} ans
        </Text>
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
          max={99}
          onChange={(v) => setPrefs((p) => ({ ...p, maxAge: v }))}
        />
      </View>

      <View style={styles.group}>
        <Text variant="caption" tone="muted" bold>
          Distance maximale
        </Text>
        <View style={styles.row}>
          {DISTANCES.map((d) => (
            <Chip
              key={d.label}
              label={d.label}
              selected={prefs.maxDistanceKm === d.value}
              onPress={() => setPrefs((p) => ({ ...p, maxDistanceKm: d.value }))}
            />
          ))}
        </View>
      </View>

      <View style={styles.switchRow}>
        <View style={styles.flex}>
          <Text bold>Afficher ma distance</Text>
          <Text variant="caption" tone="muted">
            Les autres voient une distance approximative (ex. « à 10 km »).
          </Text>
        </View>
        <Switch
          accessibilityLabel="Afficher ma distance"
          value={prefs.showDistance}
          onValueChange={(v) => setPrefs((p) => ({ ...p, showDistance: v }))}
        />
      </View>
      <Button label={submitLabel} onPress={submit} loading={saving} testID="prefs-submit" />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.xl },
  group: { gap: spacing.sm },
  row: { flexDirection: "row", gap: spacing.sm, flexWrap: "wrap" },
  switchRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  flex: { flex: 1 }
});
