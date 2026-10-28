import React from "react";
import { View } from "react-native";
import type { Preferences } from "../../api/types";
import { MIN_AGE } from "../../lib/validation";
import { spacing } from "../../theme";
import { Chip } from "../Chip";
import { Text } from "../Text";
import { GenderMultiPicker } from "./fields";

const DISTANCES = [10, 25, 50, 100, 250];

function Stepper({ label, value, min, max, onChange, testID }: { label: string; value: number; min: number; max: number; onChange: (n: number) => void; testID: string }) {
  return (
    <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}>
      <Text variant="label">{label}</Text>
      <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.sm }}>
        <Chip label="−" onPress={() => onChange(Math.max(min, value - 1))} testID={`${testID}-dec`} />
        <Text accessibilityLabel={`${label} ${value}`} style={{ minWidth: 32, textAlign: "center", fontWeight: "700" }} testID={`${testID}-value`}>
          {value}
        </Text>
        <Chip label="+" onPress={() => onChange(Math.min(max, value + 1))} testID={`${testID}-inc`} />
      </View>
    </View>
  );
}

export function PreferencesForm({ value, onChange }: { value: Preferences; onChange: (p: Preferences) => void }) {
  return (
    <View style={{ gap: spacing.xl }}>
      <GenderMultiPicker value={value.interestedIn} onChange={(interestedIn) => onChange({ ...value, interestedIn })} />
      <View style={{ gap: spacing.md }}>
        <Stepper
          label="Minimum age"
          value={value.minAge}
          min={MIN_AGE}
          max={value.maxAge}
          testID="pref-min-age"
          onChange={(minAge) => onChange({ ...value, minAge })}
        />
        <Stepper
          label="Maximum age"
          value={value.maxAge}
          min={value.minAge}
          max={99}
          testID="pref-max-age"
          onChange={(maxAge) => onChange({ ...value, maxAge })}
        />
      </View>
      <View style={{ gap: spacing.sm }}>
        <Text variant="label">Maximum distance</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {DISTANCES.map((km) => (
            <Chip
              key={km}
              label={`${km} km`}
              selected={value.maxDistanceKm === km}
              onPress={() => onChange({ ...value, maxDistanceKm: km })}
              testID={`pref-distance-${km}`}
            />
          ))}
        </View>
      </View>
    </View>
  );
}
