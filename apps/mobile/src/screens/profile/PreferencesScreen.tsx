import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { updatePreferences } from "../../api/profile";
import type { Gender } from "../../api/types";
import { setProfileCache, useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, showToast } from "../../ui";
import { MAX_AGE, MIN_AGE } from "../../utils/validation";
import { DistancePicker, InterestedInPicker, Stepper } from "./components/Pickers";

type Props = NativeStackScreenProps<AppStackParamList, "Preferences">;

export function PreferencesScreen({ navigation }: Props) {
  const { profile } = useAuth();
  const prefs = profile?.preferences;
  const [interestedIn, setInterestedIn] = useState<Gender[]>(prefs?.interestedIn ?? []);
  const [minAge, setMinAge] = useState(prefs?.minAge ?? 18);
  const [maxAge, setMaxAge] = useState(prefs?.maxAge ?? 99);
  const [distance, setDistance] = useState(prefs?.maxDistanceKm ?? 50);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    if (interestedIn.length === 0) return setError("Choisissez au moins un genre.");
    setSaving(true);
    setError(null);
    try {
      setProfileCache(await updatePreferences({ interestedIn, minAge, maxAge, maxDistanceKm: distance }));
      showToast("Préférences enregistrées. De nouveaux profils arrivent.", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen
      edges={["top", "bottom"]}
      footer={<Button label="Enregistrer" loading={saving} onPress={save} testID="preferences-save" />}
      scroll
    >
      <Header onBack={navigation.goBack} title="Préférences" />
      <View style={styles.form}>
        <InterestedInPicker onChange={setInterestedIn} value={interestedIn} />
        <View style={styles.group}>
          <Text tone="muted" variant="label">
            Tranche d’âge
          </Text>
          <Stepper label="À partir de" max={maxAge} min={MIN_AGE} onChange={setMinAge} suffix=" ans" testID="pref-min-age" value={minAge} />
          <Stepper label="Jusqu'à" max={MAX_AGE} min={minAge} onChange={setMaxAge} suffix=" ans" testID="pref-max-age" value={maxAge} />
        </View>
        <DistancePicker onChange={setDistance} value={distance} />
        {!profile?.hasLocation ? (
          <Text tone="subtle" variant="caption">
            La distance ne s’applique que si votre localisation approximative est activée (Confidentialité et sécurité).
          </Text>
        ) : null}
        {error ? <Text tone="danger">{error}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.xl, paddingTop: spacing.xs },
  group: { gap: spacing.sm }
});
