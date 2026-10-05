import { View } from "react-native";
import type { Preferences } from "../api/types";
import { FormField, Stepper } from "../shared/ui";
import { spacing } from "../theme";
import { InterestedInPicker } from "./GenderPicker";

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: ["woman", "man"], minAge: 18, maxAge: 50, maxDistanceKm: 50 };

export function PreferencesForm({ value, onChange }: { value: Preferences; onChange: (value: Preferences) => void }) {
  return (
    <View style={{ gap: spacing.xl }}>
      <FormField label="Je veux rencontrer">
        <InterestedInPicker value={value.interestedIn} onChange={(interestedIn) => onChange({ ...value, interestedIn })} />
      </FormField>
      <FormField label="Tranche d'âge">
        <Stepper
          label="Âge minimum"
          value={value.minAge}
          min={18}
          max={value.maxAge}
          format={(v) => `${v} ans`}
          onChange={(minAge) => onChange({ ...value, minAge })}
        />
        <Stepper
          label="Âge maximum"
          value={value.maxAge}
          min={value.minAge}
          max={99}
          format={(v) => `${v} ans`}
          onChange={(maxAge) => onChange({ ...value, maxAge })}
        />
      </FormField>
      <FormField label="Distance maximale" hint="Vous ne verrez que des personnes dans ce rayon (et qui vous acceptent dans le leur).">
        <Stepper
          label="Distance"
          value={value.maxDistanceKm}
          min={5}
          max={500}
          step={5}
          format={(v) => `${v} km`}
          onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
        />
      </FormField>
    </View>
  );
}
