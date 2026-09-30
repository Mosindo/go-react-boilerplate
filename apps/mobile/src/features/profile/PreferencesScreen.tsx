import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Chip, ErrorState, LoadingState, Screen, Stepper, Text } from "../../design/components";
import { errorMessage } from "../../lib/api/client";
import { profileApi } from "../../lib/api/endpoints";
import type { Gender, OwnProfile } from "../../lib/api/types";
import { interestedInLabels } from "../../lib/format";
import { queryKeys } from "../../lib/queryClient";
import { showToast } from "../../lib/toast";
import type { AppStackParamList } from "../../navigation/types";
import { useProfile, useProfileMutation } from "./hooks";

type Props = NativeStackScreenProps<AppStackParamList, "Preferences">;

export default function PreferencesScreen(props: Props) {
  const { data, isLoading, error, refetch } = useProfile();
  if (isLoading) return <LoadingState />;
  if (error || !data) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;
  return <PreferencesForm profile={data} {...props} />;
}

function PreferencesForm({ profile, navigation }: Props & { profile: OwnProfile }) {
  const client = useQueryClient();
  const [prefs, setPrefs] = useState(profile.preferences);
  const mutation = useProfileMutation(profileApi.updatePreferences);

  const toggle = (g: Gender) =>
    setPrefs((p) => ({ ...p, interestedIn: p.interestedIn.includes(g) ? p.interestedIn.filter((x) => x !== g) : [...p.interestedIn, g] }));

  const save = async () => {
    if (prefs.interestedIn.length === 0) {
      showToast("Choisissez au moins un genre.", "error");
      return;
    }
    try {
      await mutation.mutateAsync(prefs);
      // New criteria: restart discovery from scratch.
      client.removeQueries({ queryKey: queryKeys.discovery });
      showToast("Préférences enregistrées", "success");
      navigation.goBack();
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  return (
    <Screen edges={["bottom"]} footer={<Button fullWidth label="Enregistrer" loading={mutation.isPending} onPress={save} testID="prefs-save" />} scroll>
      <Text variant="heading">Je souhaite rencontrer</Text>
      <View style={styles.chips}>
        {(Object.keys(interestedInLabels) as Gender[]).map((g) => (
          <Chip key={g} label={interestedInLabels[g]} onPress={() => toggle(g)} selected={prefs.interestedIn.includes(g)} />
        ))}
      </View>
      <Text variant="heading">Tranche d’âge</Text>
      <Stepper label="Âge minimum" max={prefs.maxAge} min={18} onChange={(v) => setPrefs((p) => ({ ...p, minAge: v }))} unit="ans" value={prefs.minAge} />
      <Stepper label="Âge maximum" max={99} min={prefs.minAge} onChange={(v) => setPrefs((p) => ({ ...p, maxAge: v }))} unit="ans" value={prefs.maxAge} />
      <Text variant="heading">Distance</Text>
      <Stepper label="Distance maximale" max={500} min={5} onChange={(v) => setPrefs((p) => ({ ...p, maxDistanceKm: v }))} step={5} unit="km" value={prefs.maxDistanceKm} testID="prefs-distance" />
      {!profile.hasLocation ? (
        <Text tone="muted" variant="caption">
          Sans position enregistrée, la distance n’est pas prise en compte. Ajoutez-la depuis « Modifier mon profil ».
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
