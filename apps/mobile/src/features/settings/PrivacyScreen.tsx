import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi, safetyApi } from "../../api/endpoints";
import { keys, useBlocked, useMyProfile } from "../../hooks/queries";
import { useLocationSync } from "../../hooks/useLocation";
import { Button, Header, LoadingState, Screen, Section, Text, ToggleRow, showToast } from "../../ui";

export function PrivacyScreen() {
  const qc = useQueryClient();
  const { data } = useMyProfile();
  const blocked = useBlocked();
  const syncLocation = useLocationSync();
  const [pending, setPending] = useState(false);
  const profile = data?.profile;
  if (!profile)
    return (
      <Screen>
        <Header back title="Confidentialité" />
        <LoadingState />
      </Screen>
    );

  const update = async (isDiscoverable: boolean, showDistance: boolean) => {
    setPending(true);
    try {
      await profileApi.updatePrivacy(isDiscoverable, showDistance);
      await qc.invalidateQueries({ queryKey: keys.me });
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setPending(false);
    }
  };

  const unblock = async (userId: string) => {
    try {
      await safetyApi.unblock(userId);
      await qc.invalidateQueries({ queryKey: keys.blocked });
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  const refreshLocation = async () => {
    const outcome = await syncLocation();
    showToast(outcome === "updated" ? "Position mise à jour." : "Position indisponible.", outcome === "updated" ? "success" : "error");
  };

  return (
    <Screen scroll>
      <Header back title="Confidentialité" />
      <Section footer="En pause, votre profil n'apparaît plus dans Découvrir. Vos matchs et conversations restent accessibles." title="Visibilité">
        <ToggleRow
          description="Apparaître dans Découvrir"
          disabled={pending}
          label="Profil visible"
          onChange={(v) => update(v, profile.showDistance)}
          value={profile.isDiscoverable}
        />
        <ToggleRow
          description="Afficher une distance approximative"
          disabled={pending}
          label="Montrer la distance"
          last
          onChange={(v) => update(profile.isDiscoverable, v)}
          value={profile.showDistance}
        />
      </Section>
      <Section footer="Nous ne conservons qu'une zone d'environ 1 km. Votre position exacte n'est jamais enregistrée ni partagée." title="Localisation">
        <View style={styles.pad}>
          <Button label="Mettre à jour ma position" onPress={refreshLocation} size="md" variant="secondary" />
        </View>
      </Section>
      <Section title="Personnes bloquées">
        {blocked.isLoading ? (
          <LoadingState />
        ) : (blocked.data ?? []).length === 0 ? (
          <Text style={styles.pad} tone="muted" variant="small">
            Vous n’avez bloqué personne.
          </Text>
        ) : (
          (blocked.data ?? []).map((b) => (
            <View key={b.userId} style={styles.blockedRow}>
              <Text style={styles.flex}>{b.firstName || "Compte supprimé"}</Text>
              <Button label="Débloquer" onPress={() => unblock(b.userId)} size="md" variant="ghost" />
            </View>
          ))
        )}
      </Section>
    </Screen>
  );
}

const styles = StyleSheet.create({
  pad: { padding: 14 },
  blockedRow: { flexDirection: "row", alignItems: "center", paddingLeft: 16, paddingRight: 6, paddingVertical: 4 },
  flex: { flex: 1 }
});
