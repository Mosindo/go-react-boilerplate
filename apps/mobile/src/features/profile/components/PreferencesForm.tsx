import React from "react";
import { StyleSheet, View } from "react-native";
import { GENDER_PLURAL_LABELS, type Gender, type Preferences } from "../../../api/types";
import { spacing } from "../../../theme/palette";
import { Chip, Stepper, Text } from "../../../ui";

type Props = {
  value: Preferences;
  onChange: (next: Preferences) => void;
};

const DISTANCES = [5, 10, 25, 50, 100, 200, 500];

export function PreferencesForm({ value, onChange }: Props) {
  const toggle = (g: Gender) => {
    const has = value.interestedIn.includes(g);
    const next = has ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g];
    onChange({ ...value, interestedIn: next });
  };
  return (
    <View style={styles.wrap}>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Je souhaite rencontrer
        </Text>
        <View style={styles.chips}>
          {(Object.keys(GENDER_PLURAL_LABELS) as Gender[]).map((g) => (
            <Chip key={g} label={GENDER_PLURAL_LABELS[g]} selected={value.interestedIn.includes(g)} onPress={() => toggle(g)} testID={`interested-${g}`} />
          ))}
        </View>
        {value.interestedIn.length === 0 ? (
          <Text variant="caption" tone="danger">
            Choisis au moins une option.
          </Text>
        ) : null}
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Tranche d’âge
        </Text>
        <Stepper label="À partir de" value={value.minAge} min={18} max={value.maxAge} suffix=" ans" onChange={(minAge) => onChange({ ...value, minAge })} />
        <Stepper label="Jusqu'à" value={value.maxAge} min={value.minAge} max={99} suffix=" ans" onChange={(maxAge) => onChange({ ...value, maxAge })} />
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Distance maximale
        </Text>
        <View style={styles.chips}>
          {DISTANCES.map((d) => (
            <Chip key={d} label={`${d} km`} selected={value.maxDistanceKm === d} onPress={() => onChange({ ...value, maxDistanceKm: d })} />
          ))}
        </View>
        <Text variant="caption" tone="subtle">
          Appliquée uniquement si tu partages ta position approximative.
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.lg },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
