import React from "react";
import { StyleSheet, View } from "react-native";
import { useCurrentUser, useAuth } from "../../auth/AuthProvider";
import { useMyProfile } from "../../hooks";
import type { TabScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Avatar, ErrorState, LoadingState, Row, Screen, Section, Text } from "../../ui";
import { confirmAction } from "../../utils/confirm";

export function MeScreen({ navigation }: TabScreenProps<"Me">) {
  const user = useCurrentUser();
  const { signOut } = useAuth();
  const profile = useMyProfile();

  if (profile.isLoading)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  if (profile.isError || !profile.data)
    return (
      <Screen>
        <ErrorState onRetry={() => void profile.refetch()} />
      </Screen>
    );
  const p = profile.data;

  return (
    <Screen scroll testID="me-screen">
      <View style={styles.hero}>
        <Avatar url={p.photos[0]?.url} name={p.firstName} size={112} ring />
        <Text variant="title">
          {p.firstName}, {p.age}
        </Text>
        <Text tone="subtle" variant="label">
          {user.email}
        </Text>
        {!p.settings.discoverable ? (
          <Text tone="gold" variant="label">
            Profil en pause : tu n’apparais pas dans Découvrir.
          </Text>
        ) : null}
      </View>

      <Section title="Mon profil">
        <Row icon="create-outline" label="Modifier mon profil" testID="me-edit" onPress={() => navigation.navigate("EditProfile")} />
        <Row
          icon="images-outline"
          label="Mes photos"
          value={`${p.photos.length}/6`}
          testID="me-photos"
          onPress={() => navigation.navigate("Photos")}
        />
        <Row
          icon="options-outline"
          label="Préférences de rencontre"
          testID="me-preferences"
          onPress={() => navigation.navigate("Preferences")}
        />
      </Section>

      <Section title="Sécurité et confidentialité">
        <Row icon="lock-closed-outline" label="Confidentialité" testID="me-privacy" onPress={() => navigation.navigate("Privacy")} />
        <Row icon="ban-outline" label="Membres bloqués" testID="me-blocked" onPress={() => navigation.navigate("Blocked")} />
        <Row icon="key-outline" label="Changer de mot de passe" onPress={() => navigation.navigate("ChangePassword")} />
      </Section>

      <Section title="Compte" footer="Lueur est entièrement gratuit : aucune option payante, aucun like limité.">
        <Row
          icon="log-out-outline"
          label="Se déconnecter"
          testID="me-logout"
          onPress={async () => {
            if (await confirmAction("Se déconnecter ?", "Tu pourras te reconnecter à tout moment.", "Se déconnecter", false))
              void signOut();
          }}
        />
        <Row
          icon="trash-outline"
          label="Supprimer mon compte"
          destructive
          testID="me-delete"
          onPress={() => navigation.navigate("DeleteAccount")}
        />
      </Section>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    alignItems: "center",
    gap: spacing.sm,
    marginBottom: spacing.xl,
    marginTop: spacing.md
  }
});
