import React, { useState } from "react";
import { View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { Button, Chip, Stepper, Text } from "../ui";
import { interestedInLabels } from "../utils/format";

type Props = {
  initial: Preferences | null;
  submitLabel: string;
  loading: boolean;
  error?: string | null;
  onSubmit: (p: Preferences) => void;
};

export function PreferencesForm({ initial, submitLabel, loading, error, onSubmit }: Props) {
  const [interestedIn, setInterestedIn] = useState<Gender[]>(initial?.interestedIn ?? []);
  const [ageMin, setAgeMin] = useState(initial?.ageMin ?? 18);
  const [ageMax, setAgeMax] = useState(initial?.ageMax ?? 45);
  const [distance, setDistance] = useState(initial?.maxDistanceKm ?? 30);

  const toggle = (g: Gender) =>
    setInterestedIn((prev) => (prev.includes(g) ? prev.filter((x) => x !== g) : [...prev, g]));

  return (
    <View style={{ gap: 18 }}>
      <View style={{ gap: 8 }}>
        <Text variant="label" tone="muted">
          J’aimerais rencontrer
        </Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
          {(Object.keys(interestedInLabels) as Gender[]).map((g) => (
            <Chip
              key={g}
              testID={`interested-${g}`}
              label={interestedInLabels[g]}
              selected={interestedIn.includes(g)}
              onPress={() => toggle(g)}
            />
          ))}
        </View>
      </View>
      <Stepper label="Âge minimum" value={ageMin} min={18} max={ageMax} onChange={setAgeMin} unit="ans" />
      <Stepper label="Âge maximum" value={ageMax} min={ageMin} max={99} onChange={setAgeMax} unit="ans" />
      <Stepper label="Distance max." value={distance} min={5} max={500} step={5} onChange={setDistance} unit="km" />
      {error ? <Text tone="danger">{error}</Text> : null}
      <Button
        testID="preferences-submit"
        label={submitLabel}
        loading={loading}
        disabled={interestedIn.length === 0}
        onPress={() => onSubmit({ interestedIn, ageMin, ageMax, maxDistanceKm: distance })}
      />
    </View>
  );
}
