import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";

import { profileApi } from "../../api/endpoints";
import type { Gender, Preferences } from "../../api/types";
import { MAX_AGE, MIN_AGE } from "../../lib/validation";
import { keys } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { Screen } from "../../ui/Screen";
import { errorMessage } from "../../ui/States";
import { Text } from "../../ui/Text";

const DISTANCES: { label: string; value: number | null }[] = [
  { label: "10 km", value: 10 },
  { label: "25 km", value: 25 },
  { label: "50 km", value: 50 },
  { label: "100 km", value: 100 },
  { label: "300 km", value: 300 },
  { label: "Partout", value: null },
];
const TARGETS: { label: string; value: Gender }[] = [
  { label: "Femmes", value: "woman" },
  { label: "Hommes", value: "man" },
  { label: "Non-binaires", value: "nonbinary" },
];

function Stepper({
  label,
  value,
  min,
  max,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
}) {
  const { colors } = useTheme();
  const btn = (icon: "remove" | "add", delta: number, disabled: boolean) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${delta < 0 ? "Diminuer" : "Augmenter"} ${label}`}
      disabled={disabled}
      onPress={() => onChange(value + delta)}
      style={[
        styles.stepBtn,
        { borderColor: colors.border, backgroundColor: colors.surface },
        disabled && { opacity: 0.4 },
      ]}
    >
      <Ionicons name={icon} size={20} color={colors.text} />
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <View style={styles.stepRow}>
        {btn("remove", -1, value <= min)}
        <Text variant="heading" style={styles.stepValue}>
          {value}
        </Text>
        {btn("add", 1, value >= max)}
      </View>
    </View>
  );
}

export function PreferencesForm({
  initial,
  submitLabel,
  onSaved,
}: {
  initial: Preferences | null;
  submitLabel: string;
  onSaved: () => void;
}) {
  const qc = useQueryClient();
  const [interestedIn, setInterestedIn] = useState<Gender[]>(
    initial?.interestedIn ?? ["woman", "man", "nonbinary"],
  );
  const [minAge, setMinAge] = useState(initial?.minAge ?? MIN_AGE);
  const [maxAge, setMaxAge] = useState(initial?.maxAge ?? 60);
  const [distance, setDistance] = useState<number | null>(initial ? initial.maxDistanceKm : 50);

  const save = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (status) => {
      qc.setQueryData(keys.profile, status);
      qc.removeQueries({ queryKey: keys.discover }); // new rules: refetch candidates from scratch
      onSaved();
    },
  });

  const toggle = (g: Gender) =>
    setInterestedIn((cur) =>
      cur.includes(g) ? (cur.length > 1 ? cur.filter((x) => x !== g) : cur) : [...cur, g],
    );

  return (
    <>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Je souhaite rencontrer
        </Text>
        <View style={styles.wrap}>
          {TARGETS.map((t) => (
            <Chip
              key={t.value}
              label={t.label}
              selected={interestedIn.includes(t.value)}
              onPress={() => toggle(t.value)}
            />
          ))}
        </View>
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Tranche d&apos;âge
        </Text>
        <View style={styles.stepBoth}>
          <Stepper label="Min." value={minAge} min={MIN_AGE} max={maxAge} onChange={setMinAge} />
          <Stepper label="Max." value={maxAge} min={minAge} max={MAX_AGE} onChange={setMaxAge} />
        </View>
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Distance maximale
        </Text>
        <View style={styles.wrap}>
          {DISTANCES.map((d) => (
            <Chip
              key={d.label}
              label={d.label}
              selected={distance === d.value}
              onPress={() => setDistance(d.value)}
            />
          ))}
        </View>
      </View>
      {save.isError ? <Text tone="danger">{errorMessage(save.error)}</Text> : null}
      <Button
        label={submitLabel}
        loading={save.isPending}
        onPress={() => save.mutate({ interestedIn, minAge, maxAge, maxDistanceKm: distance })}
        testID="prefs-submit"
      />
    </>
  );
}

export function PreferencesSetupScreen({
  initial,
  onDone,
}: {
  initial: Preferences | null;
  onDone: () => void;
}) {
  return (
    <Screen scroll>
      <Text variant="title">Vos préférences</Text>
      <Text tone="muted">Étape 2 sur 3. Vous pourrez les modifier à tout moment.</Text>
      <PreferencesForm initial={initial} submitLabel="Continuer" onSaved={onDone} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  group: { gap: spacing.sm },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  stepBoth: { flexDirection: "row", gap: spacing.xl },
  stepper: { gap: spacing.xs },
  stepRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  stepBtn: {
    width: 40,
    height: 40,
    borderRadius: radii.pill,
    borderWidth: 1.5,
    alignItems: "center",
    justifyContent: "center",
  },
  stepValue: { minWidth: 36, textAlign: "center" },
});
