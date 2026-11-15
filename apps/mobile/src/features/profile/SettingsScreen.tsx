import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../lib/errors";
import type { AppStackParamList } from "../../navigation/types";
import { useSession } from "../../session/SessionProvider";
import { ConfirmSheet, Header, ListRow, Loading, Screen, Section, Text, toast } from "../../ui";
import { useMyProfile, useUpdatePrivacy } from "./hooks";

type Props = NativeStackScreenProps<AppStackParamList, "Settings">;

export function SettingsScreen({ navigation }: Props) {
  const { user, signOut } = useSession();
  const me = useMyProfile();
  const privacy = useUpdatePrivacy();
  const [confirmLogout, setConfirmLogout] = useState(false);

  if (!me.data) return <Loading />;
  const { isPaused, hideDistance } = me.data.profile;
  const setPrivacy = (next: { isPaused: boolean; hideDistance: boolean }) =>
    privacy.mutate(next, {
      onSuccess: () => toast("Confidentialité mise à jour", "success"),
      onError: (e) => toast(errorMessage(e), "error")
    });

  return (
    <Screen scroll edges={["top", "bottom"]} testID="settings-screen">
      <Header title="Réglages" onBack={() => navigation.goBack()} />
      <Section title="Confidentialité" footer="Votre position exacte n'est jamais enregistrée : seule une zone d'environ 1 km l'est.">
        <ListRow
          icon="pause-circle-outline"
          label="Mettre mon profil en pause"
          description="Vous n'apparaissez plus dans la découverte. Vos matchs restent actifs."
          toggle={{ value: isPaused, onChange: (v) => setPrivacy({ isPaused: v, hideDistance }), disabled: privacy.isPending }}
          testID="toggle-pause"
        />
        <ListRow
          icon="eye-off-outline"
          label="Masquer ma distance"
          description="Les autres ne voient pas à quelle distance vous êtes."
          toggle={{ value: hideDistance, onChange: (v) => setPrivacy({ isPaused, hideDistance: v }), disabled: privacy.isPending }}
          testID="toggle-hide-distance"
        />
        <ListRow icon="ban-outline" label="Personnes bloquées" onPress={() => navigation.navigate("BlockedUsers")} testID="open-blocked" />
      </Section>
      <Section title="Compte">
        <ListRow icon="mail-outline" label="E-mail" value={user?.email} />
        <ListRow icon="key-outline" label="Changer de mot de passe" onPress={() => navigation.navigate("ChangePassword")} />
        <ListRow icon="log-out-outline" label="Se déconnecter" onPress={() => setConfirmLogout(true)} testID="logout" />
        <ListRow
          icon="trash-outline"
          label="Supprimer mon compte"
          tone="danger"
          onPress={() => navigation.navigate("DeleteAccount")}
          testID="open-delete-account"
        />
      </Section>
      <Text variant="caption" color="textSubtle" align="center">
        Lueur — gratuite, sans publicité, sans abonnement.
      </Text>
      <ConfirmSheet
        visible={confirmLogout}
        title="Se déconnecter ?"
        message="Vous pourrez vous reconnecter à tout moment avec votre e-mail."
        confirmLabel="Se déconnecter"
        onConfirm={() => void signOut()}
        onClose={() => setConfirmLogout(false)}
      />
    </Screen>
  );
}
