import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Preferences } from "../../api/types";
import { useMyProfile } from "../../hooks/queries";
import { keys, queryClient } from "../../hooks/queryClient";
import { spacing } from "../../theme/palette";
import { Button, Header, InlineError, Loading, Screen, Section, toast } from "../../ui";
import type { AppScreenProps } from "../../navigation/types";
import { LocationCard } from "./components/LocationCard";
import { PreferencesForm } from "./components/PreferencesForm";

function Form({ initial, hasLocation, onSaved }: { initial: Preferences; hasLocation: boolean; onSaved: () => void }) {
  const [prefs, setPrefs] = useState(initial);
  const save = useMutation({
    mutationFn: () => profileApi.savePreferences(prefs),
    onSuccess: (profile) => {
      queryClient.setQueryData(keys.myProfile, profile);
      toast("Critères mis à jour.", "success");
      onSaved();
    }
  });
  return (
    <View style={styles.content}>
      <PreferencesForm value={prefs} onChange={setPrefs} />
      <Section title="Position">
        <LocationCard hasLocation={hasLocation} />
      </Section>
      <InlineError message={save.error ? errorMessage(save.error) : null} />
      <Button label="Enregistrer" size="lg" disabled={prefs.interestedIn.length === 0} loading={save.isPending} onPress={() => save.mutate()} style={styles.save} testID="preferences-save" />
    </View>
  );
}

export function PreferencesScreen({ navigation }: AppScreenProps<"Preferences">) {
  const profile = useMyProfile();
  return (
    <Screen scroll>
      <Header title="Critères de recherche" onBack={() => navigation.goBack()} />
      {profile.data ? (
        <Form initial={profile.data.preferences} hasLocation={profile.data.hasLocation} onSaved={() => navigation.goBack()} />
      ) : (
        <Loading />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 560, width: "100%", alignSelf: "center", paddingTop: spacing.sm },
  save: { marginTop: spacing.xl }
});
