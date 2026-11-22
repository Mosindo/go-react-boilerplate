import React from "react";
import { useAuth } from "../../hooks/useAuth";
import { confirm } from "../../lib/confirm";
import type { MainScreenProps } from "../../navigation/types";
import { Header, ListRow, Screen, Section, Text } from "../../ui";

export default function SettingsScreen({ navigation }: MainScreenProps<"Settings">) {
  const { user, signOut } = useAuth();

  return (
    <Screen scroll testID="settings-screen">
      <Header onBack={navigation.goBack} subtitle={user?.email} title="Réglages" />
      <Section title="Sécurité">
        <ListRow
          icon="key-outline"
          onPress={() => navigation.navigate("ChangePassword")}
          subtitle="Déconnecte vos autres appareils"
          title="Changer de mot de passe"
        />
        <ListRow
          icon="ban-outline"
          onPress={() => navigation.navigate("BlockedUsers")}
          testID="settings-blocked"
          title="Personnes bloquées"
        />
      </Section>
      <Section title="Session">
        <ListRow
          icon="log-out-outline"
          onPress={async () => {
            if (await confirm("Se déconnecter ?", "Vous pourrez vous reconnecter à tout moment.", "Se déconnecter")) {
              await signOut();
            }
          }}
          testID="settings-logout"
          title="Se déconnecter"
        />
      </Section>
      <Section title="Zone sensible">
        <ListRow
          destructive
          icon="trash-outline"
          onPress={() => navigation.navigate("DeleteAccount")}
          testID="settings-delete"
          title="Supprimer mon compte"
        />
      </Section>
      <Text align="center" tone="subtle" variant="caption">
        Une question de sécurité ? Signalez tout comportement inapproprié depuis le profil concerné.
      </Text>
    </Screen>
  );
}
