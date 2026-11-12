import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../api/types";
import { spacing } from "../theme";
import { Chip, Stepper, Text } from "../ui";
import { seekingLabels } from "../utils/format";

const genders: Gender[] = ["woman", "man", "nonbinary"];

export function PreferencesForm({ value, onChange }: { value: Preferences; onChange: (p: Preferences) => void }) {
  const toggle = (g: Gender) => {
    const next = value.interestedIn.includes(g) ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g];
    onChange({ ...value, interestedIn: next });
  };
  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">
        Je souhaite rencontrer
      </Text>
      <View style={styles.chips}>
        {genders.map((g) => (
          <Chip key={g} label={seekingLabels[g]} selected={value.interestedIn.includes(g)} onPress={() => toggle(g)} />
        ))}
      </View>
      <Stepper label="Âge minimum" value={value.minAge} min={18} max={value.maxAge} onChange={(minAge) => onChange({ ...value, minAge })} />
      <Stepper label="Âge maximum" value={value.maxAge} min={value.minAge} max={99} onChange={(maxAge) => onChange({ ...value, maxAge })} />
      <Stepper
        label="Distance maximale"
        value={value.maxDistanceKm}
        min={5}
        max={500}
        step={value.maxDistanceKm >= 100 ? 50 : 5}
        format={(v) => `${v} km`}
        onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, marginBottom: spacing.xs }
});
