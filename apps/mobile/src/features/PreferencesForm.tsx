import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import { GENDER_LABELS, type Gender, type Preferences } from "../api/types";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import { Button, Chip, Stepper, Text, useToast } from "../ui";

const DISTANCES: { label: string; value: number | null }[] = [
  { label: "10 km", value: 10 },
  { label: "25 km", value: 25 },
  { label: "50 km", value: 50 },
  { label: "100 km", value: 100 },
  { label: "300 km", value: 300 },
  { label: "Partout", value: null }
];

type Props = { initial?: Preferences | null; submitLabel: string; onSaved: () => void };

export function PreferencesForm({ initial, submitLabel, onSaved }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const [interestedIn, setInterestedIn] = useState<Gender[]>(initial?.interestedIn ?? []);
  const [minAge, setMinAge] = useState(initial?.minAge ?? 18);
  const [maxAge, setMaxAge] = useState(initial?.maxAge ?? 45);
  const [distance, setDistance] = useState<number | null>(initial ? initial.maxDistanceKm : 50);

  const save = useMutation({
    mutationFn: () =>
      profileApi.savePreferences({ interestedIn, minAge, maxAge, maxDistanceKm: distance }),
    onSuccess: (overview) => {
      qc.setQueryData(queryKeys.overview, overview);
      // The feed depends on preferences: drop the stale deck.
      qc.removeQueries({ queryKey: queryKeys.feed });
      onSaved();
    },
    onError: (err) => toast.show(errorMessage(err), "error")
  });

  const toggle = (g: Gender) =>
    setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]));

  return (
    <View style={styles.form}>
      <View style={styles.group}>
        <Text variant="label">Je souhaite rencontrer</Text>
        <View style={styles.wrap}>
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
        <Text variant="label">Tranche d'âge</Text>
        <Stepper label="Âge minimum" value={minAge} min={18} max={maxAge} onChange={setMinAge} />
        <Stepper label="Âge maximum" value={maxAge} min={minAge} max={99} onChange={setMaxAge} />
      </View>
      <View style={styles.group}>
        <Text variant="label">Distance maximale</Text>
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
        <Text variant="caption" tone="muted">
          Un filtre de distance suppose que vous partagez votre position approximative.
        </Text>
      </View>
      <Button
        label={submitLabel}
        loading={save.isPending}
        disabled={interestedIn.length === 0}
        onPress={() => save.mutate()}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: 22 },
  group: { gap: 10 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
