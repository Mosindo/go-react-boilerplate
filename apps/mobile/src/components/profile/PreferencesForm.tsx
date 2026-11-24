import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../../api/types";
import { colors, spacing } from "../../theme/tokens";
import { genderPluralLabels } from "../../utils/format";
import { AppText } from "../AppText";
import { Chip } from "../Chip";
import { Stepper } from "../Stepper";

const genders: Gender[] = ["woman", "man", "nonbinary"];

export function PreferencesForm({ value, onChange }: { value: Preferences; onChange: (p: Preferences) => void }) {
  const toggle = (g: Gender) => {
    const has = value.interestedIn.includes(g);
    const next = has ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g];
    if (next.length > 0) onChange({ ...value, interestedIn: next });
  };
  return (
    <View style={styles.form}>
      <View style={styles.group}>
        <AppText variant="label" color={colors.textMuted}>
          Je veux rencontrer
        </AppText>
        <View style={styles.chips}>
          {genders.map((g) => (
            <Chip key={g} label={genderPluralLabels[g]} selected={value.interestedIn.includes(g)} onPress={() => toggle(g)} testID={`prefs-gender-${g}`} />
          ))}
        </View>
      </View>
      <View style={styles.group}>
        <AppText variant="label" color={colors.textMuted}>
          Âge
        </AppText>
        <Stepper label="À partir de" value={value.minAge} min={18} max={value.maxAge} unit="ans" onChange={(minAge) => onChange({ ...value, minAge })} testID="prefs-min-age" />
        <Stepper label="Jusqu'à" value={value.maxAge} min={value.minAge} max={99} unit="ans" onChange={(maxAge) => onChange({ ...value, maxAge })} testID="prefs-max-age" />
      </View>
      <View style={styles.group}>
        <AppText variant="label" color={colors.textMuted}>
          Distance maximale
        </AppText>
        <Stepper
          label="Rayon"
          value={value.maxDistanceKm}
          min={5}
          max={300}
          step={5}
          unit="km"
          onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
          testID="prefs-distance"
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
