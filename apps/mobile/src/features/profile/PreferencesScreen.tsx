import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { profileApi } from "../../api/profile";
import type { Gender, Preferences } from "../../api/types";
import type { RootScreenProps } from "../../navigation/types";
import { useMyProfile, useProfileMutation } from "../../state/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, Header, LoadingState, Screen, Stepper, Text, useToast } from "../../ui";
import { describeError } from "../../utils/errors";
import { interestedInLabels } from "../../utils/format";

export default function PreferencesScreen({ navigation }: RootScreenProps<"Preferences">) {
  const { data } = useMyProfile();
  if (!data) return <LoadingState />;
  return <PreferencesForm initial={data.preferences} onDone={() => navigation.goBack()} />;
}

function PreferencesForm({ initial, onDone }: { initial: Preferences; onDone: () => void }) {
  const { spacing, colors, radii } = useTheme();
  const toast = useToast();
  const [prefs, setPrefs] = useState<Preferences>(initial);
  const save = useProfileMutation((p: Preferences) => profileApi.updatePreferences(p));
  const toggle = (g: Gender) =>
    setPrefs((p) => ({
      ...p,
      interestedIn: p.interestedIn.includes(g) ? p.interestedIn.filter((x) => x !== g) : [...p.interestedIn, g]
    }));

  return (
    <Screen
      scroll
      testID="preferences-screen"
      footer={
        <Button
          label="Enregistrer"
          disabled={prefs.interestedIn.length === 0}
          loading={save.isPending}
          onPress={() =>
            save.mutate(prefs, {
              onSuccess: () => {
                toast.show("Préférences enregistrées.", "success");
                onDone();
              }
            })
          }
          testID="preferences-save"
        />
      }
    >
      <Header back title="Préférences" />
      <View style={{ gap: spacing.xl }}>
        <View style={{ gap: spacing.sm }}>
          <Text variant="heading">Je souhaite rencontrer</Text>
          <View style={styles.chips}>
            {(Object.keys(interestedInLabels) as Gender[]).map((g) => (
              <Chip
                key={g}
                label={interestedInLabels[g]}
                selected={prefs.interestedIn.includes(g)}
                onPress={() => toggle(g)}
              />
            ))}
          </View>
          {prefs.interestedIn.length === 0 ? <Text tone="danger">Choisissez au moins une option.</Text> : null}
        </View>
        <View
          style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.lg }]}
        >
          <Text variant="heading">Âge</Text>
          <Stepper
            label="Minimum"
            value={prefs.ageMin}
            min={18}
            max={prefs.ageMax}
            unit=" ans"
            onChange={(ageMin) => setPrefs((p) => ({ ...p, ageMin }))}
            testID="pref-age-min"
          />
          <Stepper
            label="Maximum"
            value={prefs.ageMax}
            min={prefs.ageMin}
            max={99}
            unit=" ans"
            onChange={(ageMax) => setPrefs((p) => ({ ...p, ageMax }))}
            testID="pref-age-max"
          />
        </View>
        <View
          style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.lg }]}
        >
          <Text variant="heading">Distance</Text>
          <Stepper
            label="Jusqu'à"
            value={prefs.maxDistanceKm}
            min={5}
            max={500}
            step={5}
            unit=" km"
            onChange={(maxDistanceKm) => setPrefs((p) => ({ ...p, maxDistanceKm }))}
            testID="pref-distance"
          />
          <Text variant="caption" tone="subtle">
            Les profils doivent aussi vous inclure dans leurs propres critères pour apparaître.
          </Text>
        </View>
        {save.error ? <Text tone="danger">{describeError(save.error)}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  card: { padding: 16, gap: 4, borderWidth: StyleSheet.hairlineWidth }
});
