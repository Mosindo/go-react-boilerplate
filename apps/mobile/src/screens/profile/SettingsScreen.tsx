import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/profile";
import { keys, useMyProfile } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { ConfirmDialog, Header, ListGroup, ListRow, Screen, Text, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { LocationDeniedError, shareApproximateLocation } from "../../utils/location";

export default function SettingsScreen({ navigation }: AppScreenProps<"Settings">) {
  const qc = useQueryClient();
  const me = useMyProfile();
  const { user, signOut } = useAuth();
  const [confirmLogout, setConfirmLogout] = useState(false);
  const [locating, setLocating] = useState(false);
  const discoverable = me.data?.profile?.isDiscoverable ?? true;

  const toggleVisibility = async (value: boolean) => {
    try {
      await profileApi.setVisibility(value);
      await qc.invalidateQueries({ queryKey: keys.me });
      showToast(value ? "Ton profil est de nouveau visible" : "Profil en pause : tu n'apparais plus dans la découverte", "success");
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  const updateLocation = async () => {
    setLocating(true);
    try {
      await shareApproximateLocation();
      await qc.invalidateQueries({ queryKey: keys.me });
      void qc.invalidateQueries({ queryKey: keys.deck });
      showToast("Position approximative mise à jour", "success");
    } catch (err) {
      showToast(err instanceof LocationDeniedError ? err.message : "Impossible d'obtenir ta position.", "error");
    } finally {
      setLocating(false);
    }
  };

  return (
    <Screen scroll testID="settings-screen">
      <Header title="Confidentialité et compte" onBack={navigation.goBack} />
      <View style={styles.groups}>
        <Text variant="label" tone="muted">
          CONFIDENTIALITÉ
        </Text>
        <ListGroup>
          <ListRow
            icon="eye-outline"
            title="Profil visible"
            subtitle="Désactive pour faire une pause : tes matchs et conversations restent accessibles."
            value={discoverable}
            onValueChange={toggleVisibility}
            testID="settings-visibility"
          />
          <ListRow
            icon="navigate-outline"
            title={locating ? "Localisation…" : "Mettre à jour ma position"}
            subtitle={me.data?.completeness.hasLocation ? "Position approximative (~1 km) enregistrée" : "Aucune position partagée"}
            onPress={locating ? undefined : updateLocation}
          />
          <ListRow icon="ban-outline" title="Personnes bloquées" onPress={() => navigation.navigate("BlockedUsers")} testID="settings-blocked" />
        </ListGroup>

        <Text variant="label" tone="muted">
          COMPTE
        </Text>
        <ListGroup>
          <ListRow icon="mail-outline" title="Email" subtitle={user?.email} />
          <ListRow icon="key-outline" title="Changer de mot de passe" onPress={() => navigation.navigate("ChangePassword")} />
          <ListRow icon="log-out-outline" title="Se déconnecter" onPress={() => setConfirmLogout(true)} testID="settings-logout" />
          <ListRow
            icon="trash-outline"
            title="Supprimer mon compte"
            tone="danger"
            onPress={() => navigation.navigate("DeleteAccount")}
            testID="settings-delete"
          />
        </ListGroup>

        <Text variant="caption" tone="subtle">
          Tes données ne sont jamais vendues. Ton email, ta date de naissance et ta position ne sont jamais montrés aux autres membres.
        </Text>
      </View>
      <ConfirmDialog
        visible={confirmLogout}
        title="Se déconnecter ?"
        message="Tu pourras te reconnecter à tout moment."
        confirmLabel="Se déconnecter"
        onCancel={() => setConfirmLogout(false)}
        onConfirm={() => {
          setConfirmLogout(false);
          void signOut();
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({ groups: { gap: spacing.md } });
