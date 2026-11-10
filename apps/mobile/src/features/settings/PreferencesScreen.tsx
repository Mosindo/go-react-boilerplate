import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender } from "../../api/types";
import { keys, useMyProfile } from "../../hooks/queries";
import type { MainScreenProps } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, Header, IconButton, LoadingState, Screen, Section, Text, showToast } from "../../ui";
import { interestedInLabels } from "../../utils/format";

const GENDERS: Gender[] = ["woman", "man", "nonbinary"];
const DISTANCES = [5, 10, 25, 50, 100, 200, 500];

function Stepper({ label, value, min, max, onChange }: { label: string; value: number; min: number; max: number; onChange: (v: number) => void }) {
  const theme = useTheme();
  return (
    <View style={styles.stepper}>
      <Text style={styles.flex}>{label}</Text>
      <IconButton disabled={value <= min} icon="remove" label={`Diminuer ${label}`} onPress={() => onChange(Math.max(min, value - 1))} size={36} />
      <Text accessibilityLabel={`${label} : ${value} ans`} style={[styles.value, { color: theme.colors.text }]} variant="heading">
        {value}
      </Text>
      <IconButton disabled={value >= max} icon="add" label={`Augmenter ${label}`} onPress={() => onChange(Math.min(max, value + 1))} size={36} />
    </View>
  );
}

export function PreferencesScreen({ navigation }: MainScreenProps<"Preferences">) {
  const qc = useQueryClient();
  const { data } = useMyProfile();
  const prefs = data?.preferences;
  const [interestedIn, setInterestedIn] = useState<Gender[]>([]);
  const [minAge, setMinAge] = useState(18);
  const [maxAge, setMaxAge] = useState(99);
  const [distance, setDistance] = useState(50);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!prefs) return;
    setInterestedIn(prefs.interestedIn);
    setMinAge(prefs.minAge);
    setMaxAge(prefs.maxAge);
    setDistance(prefs.maxDistanceKm);
  }, [prefs]);

  if (!prefs)
    return (
      <Screen>
        <Header back title="Préférences" />
        <LoadingState />
      </Screen>
    );

  const save = async () => {
    if (interestedIn.length === 0) {
      showToast("Choisissez au moins une option.", "error");
      return;
    }
    setSaving(true);
    try {
      await profileApi.updatePreferences({ interestedIn, minAge, maxAge, maxDistanceKm: distance });
      await qc.invalidateQueries({ queryKey: keys.me });
      showToast("Préférences enregistrées.", "success");
      navigation.goBack();
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen scroll footer={<Button label="Enregistrer" loading={saving} onPress={save} testID="prefs-save" />}>
      <Header back title="Préférences de rencontre" />
      <Section title="Je souhaite rencontrer">
        <View style={styles.chips}>
          {GENDERS.map((g) => (
            <Chip
              key={g}
              label={interestedInLabels[g]}
              onPress={() => setInterestedIn((v) => (v.includes(g) ? v.filter((x) => x !== g) : [...v, g]))}
              selected={interestedIn.includes(g)}
            />
          ))}
        </View>
      </Section>
      <Section footer="Les personnes doivent aussi correspondre à vos critères pour que vous vous découvriez mutuellement." title="Âge">
        <Stepper label="Âge minimum" max={maxAge} min={18} onChange={setMinAge} value={minAge} />
        <Stepper label="Âge maximum" max={99} min={minAge} onChange={setMaxAge} value={maxAge} />
      </Section>
      <Section title="Distance maximale">
        <View style={styles.chips}>
          {DISTANCES.map((d) => (
            <Chip key={d} label={`${d} km`} onPress={() => setDistance(d)} selected={distance === d} />
          ))}
        </View>
      </Section>
    </Screen>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8, padding: 14 },
  stepper: { flexDirection: "row", alignItems: "center", gap: 8, paddingHorizontal: 16, paddingVertical: 8 },
  flex: { flex: 1 },
  value: { minWidth: 36, textAlign: "center" }
});
