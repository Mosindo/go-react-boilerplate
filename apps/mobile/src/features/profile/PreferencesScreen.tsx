import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Gender, Preferences } from "../../api/types";
import { interestedInLabels } from "../../lib/format";
import { getApproximateLocation } from "../../lib/location";
import type { AppStackParamList } from "../../navigation/types";
import { useSession, useSetProfile } from "../../state/session";
import { spacing } from "../../theme/tokens";
import { Button, Card, Header, ListRow, LoadingState, Screen, SectionTitle, Stepper, Text, ToggleRow, useFeedback } from "../../ui";
import { ChoiceList } from "./ChoiceList";

type Props = NativeStackScreenProps<AppStackParamList, "Preferences">;

export function PreferencesScreen({ navigation }: Props) {
  const { profile } = useSession();
  if (!profile) return <LoadingState />;
  return <PreferencesForm navigation={navigation} initial={profile.preferences} />;
}

function PreferencesForm({ navigation, initial }: { navigation: Props["navigation"]; initial: Preferences }) {
  const { profile } = useSession();
  const setProfile = useSetProfile();
  const client = useQueryClient();
  const { toast } = useFeedback();
  const [prefs, setPrefs] = useState<Preferences>(initial);
  const patch = (p: Partial<Preferences>) => setPrefs((cur) => ({ ...cur, ...p }));

  const refreshDiscovery = () => {
    void client.invalidateQueries({ queryKey: queryKeys.discovery });
  };

  const save = useMutation({
    mutationFn: () => profileApi.savePreferences(prefs),
    onSuccess: (saved) => {
      if (profile) setProfile({ ...profile, preferences: saved });
      refreshDiscovery();
      toast("Préférences enregistrées", "success");
      navigation.goBack();
    }
  });

  const locate = useMutation({
    mutationFn: async () => {
      const loc = await getApproximateLocation();
      await profileApi.saveLocation(loc.latitude, loc.longitude, loc.city);
      return profileApi.get();
    },
    onSuccess: (p) => {
      setProfile(p);
      refreshDiscovery();
      toast("Position approximative mise à jour", "success");
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const forget = useMutation({
    mutationFn: async () => {
      await profileApi.clearLocation();
      return profileApi.get();
    },
    onSuccess: (p) => {
      setProfile(p);
      refreshDiscovery();
      toast("Position supprimée", "success");
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const toggleGender = (g: Gender) =>
    patch({ interestedIn: prefs.interestedIn.includes(g) ? prefs.interestedIn.filter((x) => x !== g) : [...prefs.interestedIn, g] });

  return (
    <Screen
      scroll
      footer={
        <View style={styles.footer}>
          {save.error ? <Text tone="danger">{errorMessage(save.error)}</Text> : null}
          <Button
            label="Enregistrer"
            onPress={() => save.mutate()}
            loading={save.isPending}
            disabled={prefs.interestedIn.length === 0}
            testID="prefs-save"
          />
        </View>
      }
    >
      <Header title="Préférences" onBack={navigation.goBack} />
      <SectionTitle>Je souhaite rencontrer</SectionTitle>
      <ChoiceList
        multiple
        options={(Object.keys(interestedInLabels) as Gender[]).map((g) => ({ value: g, label: interestedInLabels[g] }))}
        selected={prefs.interestedIn}
        onToggle={toggleGender}
      />
      <SectionTitle>Âge et distance</SectionTitle>
      <Card>
        <Stepper label="Âge minimum" value={prefs.ageMin} min={18} max={prefs.ageMax} onChange={(v) => patch({ ageMin: v })} />
        <Stepper label="Âge maximum" value={prefs.ageMax} min={prefs.ageMin} max={99} onChange={(v) => patch({ ageMax: v })} />
        <Stepper
          label="Distance max."
          value={prefs.maxDistanceKm}
          min={5}
          max={500}
          step={5}
          suffix=" km"
          onChange={(v) => patch({ maxDistanceKm: v })}
        />
      </Card>
      <SectionTitle>Localisation</SectionTitle>
      <Card>
        <ListRow
          icon="navigate-outline"
          label={profile?.hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"}
          value={profile?.city || undefined}
          onPress={() => locate.mutate()}
        />
        {profile?.hasLocation ? (
          <ListRow icon="close-circle-outline" label="Supprimer ma position" onPress={() => forget.mutate()} />
        ) : null}
      </Card>
      <Text variant="caption" tone="subtle" style={styles.note}>
        Seule une position arrondie (environ 1 km) est conservée. Les autres membres voient une distance approximative.
      </Text>
      <SectionTitle>Confidentialité</SectionTitle>
      <Card>
        <ToggleRow
          label="Apparaître dans la découverte"
          description="Désactivez pour faire une pause : vos matchs et conversations restent accessibles."
          value={prefs.isDiscoverable}
          onChange={(v) => patch({ isDiscoverable: v })}
          testID="prefs-discoverable"
        />
        <ToggleRow label="Afficher ma distance" value={prefs.showDistance} onChange={(v) => patch({ showDistance: v })} />
      </Card>
    </Screen>
  );
}

const styles = StyleSheet.create({
  footer: { gap: spacing.xs, paddingVertical: spacing.xs },
  note: { marginTop: spacing.xs, marginHorizontal: spacing.xs }
});
