import React from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { useMyProfile } from "../../hooks/queries";
import { keys, queryClient } from "../../hooks/queryClient";
import { Card, Header, ListRow, Loading, Screen, Section, Text, ToggleRow, toast } from "../../ui";
import type { AppScreenProps } from "../../navigation/types";
import { LocationCard } from "./components/LocationCard";

export function PrivacyScreen({ navigation }: AppScreenProps<"Privacy">) {
  const profile = useMyProfile();
  const update = useMutation({
    mutationFn: (input: { isVisible: boolean; showDistance: boolean }) => profileApi.setPrivacy(input.isVisible, input.showDistance),
    onSuccess: (p) => queryClient.setQueryData(keys.myProfile, p),
    onError: (e) => toast(errorMessage(e), "error")
  });
  const p = profile.data;

  return (
    <Screen scroll>
      <Header title="Confidentialité" onBack={() => navigation.goBack()} />
      {!p ? (
        <Loading />
      ) : (
        <View style={styles.content}>
          <Section title="Visibilité">
            <Card>
              <ToggleRow
                title="Apparaître dans Découvrir"
                subtitle="Désactive pour mettre ton profil en pause. Tes matchs et conversations restent accessibles."
                value={p.isVisible}
                disabled={update.isPending}
                onChange={(isVisible) => update.mutate({ isVisible, showDistance: p.showDistance })}
                testID="privacy-visible"
              />
              <ToggleRow
                title="Afficher ma distance"
                subtitle="Si désactivé, les autres ne voient pas à quelle distance tu te trouves."
                value={p.showDistance}
                disabled={update.isPending}
                onChange={(showDistance) => update.mutate({ isVisible: p.isVisible, showDistance })}
              />
            </Card>
          </Section>
          <Section title="Position">
            <LocationCard hasLocation={p.hasLocation} />
          </Section>
          <Section title="Sécurité">
            <Card>
              <ListRow icon="ban-outline" title="Personnes bloquées" onPress={() => navigation.navigate("BlockedUsers")} />
            </Card>
          </Section>
          <Section title="Tes données">
            <Text variant="caption" tone="muted">
              Ton e-mail, ta date de naissance exacte et ta position ne sont jamais montrés aux autres membres. Les photos sont servies par des
              liens temporaires signés. Tu peux supprimer ton compte et toutes tes données à tout moment depuis l’onglet « Moi ».
            </Text>
          </Section>
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 560, width: "100%", alignSelf: "center" }
});
