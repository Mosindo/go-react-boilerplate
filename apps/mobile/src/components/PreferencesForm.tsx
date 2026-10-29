import { useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { savePreferences } from "../api/profile";
import type { Gender, Preferences } from "../api/types";
import { bumpDiscoverVersion } from "../state/discoverStore";
import { queryKeys } from "../hooks/queries";
import { useAsyncAction } from "../hooks/useAsyncAction";
import { GENDER_LABELS } from "../lib/format";
import { fontSizes, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { Chip } from "./Chip";
import { Stepper } from "./Stepper";

const DISTANCES = [10, 25, 50, 100, 250, 500];
const DEFAULTS: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 };

type Props = {
  initial: Preferences | null;
  submitLabel: string;
  onSaved: (preferences: Preferences) => void;
};

export function PreferencesForm({ initial, submitLabel, onSaved }: Props) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const start = initial ?? DEFAULTS;
  const [interestedIn, setInterestedIn] = useState<Gender[]>(start.interestedIn);
  const [minAge, setMinAge] = useState(start.minAge);
  const [maxAge, setMaxAge] = useState(start.maxAge);
  const [distance, setDistance] = useState(start.maxDistanceKm);

  const distances = DISTANCES.includes(distance)
    ? DISTANCES
    : [...DISTANCES, distance].sort((a, b) => a - b);

  const toggle = (gender: Gender) =>
    setInterestedIn((current) =>
      current.includes(gender) ? current.filter((g) => g !== gender) : [...current, gender]
    );

  const submit = useAsyncAction(async () => {
    if (interestedIn.length === 0) {
      throw new Error("Choisissez au moins une option.");
    }
    const saved = await savePreferences({ interestedIn, minAge, maxAge, maxDistanceKm: distance });
    queryClient.setQueryData(queryKeys.preferences, saved);
    bumpDiscoverVersion();
    onSaved(saved);
  });

  return (
    <View>
      <Text style={[styles.label, { color: theme.textMuted }]}>Je souhaite rencontrer</Text>
      <View style={styles.chips}>
        {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
          <Chip
            key={g}
            label={GENDER_LABELS[g]}
            onPress={() => toggle(g)}
            selected={interestedIn.includes(g)}
          />
        ))}
      </View>

      <Text style={[styles.label, styles.section, { color: theme.textMuted }]}>
        Tranche d&apos;âge
      </Text>
      <Stepper
        label="Âge minimum"
        max={Math.min(maxAge, 99)}
        min={18}
        onChange={setMinAge}
        suffix=" ans"
        value={minAge}
      />
      <Stepper
        label="Âge maximum"
        max={99}
        min={Math.max(minAge, 18)}
        onChange={setMaxAge}
        suffix=" ans"
        value={maxAge}
      />

      <Text style={[styles.label, styles.section, { color: theme.textMuted }]}>
        Distance maximale
      </Text>
      <View style={styles.chips}>
        {distances.map((km) => (
          <Chip
            key={km}
            label={`${km} km`}
            onPress={() => setDistance(km)}
            selected={distance === km}
          />
        ))}
      </View>

      {submit.error ? (
        <Text style={[styles.error, { color: theme.danger }]}>{submit.error}</Text>
      ) : null}
      <Button
        loading={submit.loading}
        onPress={() => void submit.run()}
        style={styles.submit}
        testID="preferences-submit"
        title={submitLabel}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  label: {
    fontSize: fontSizes.xs,
    fontWeight: "700",
    letterSpacing: 0.6,
    marginBottom: spacing.sm,
    textTransform: "uppercase"
  },
  section: { marginTop: spacing.xl },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  error: { fontSize: fontSizes.sm, marginTop: spacing.md },
  submit: { marginTop: spacing.xl }
});
