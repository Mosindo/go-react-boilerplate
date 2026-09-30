import React, { useState } from "react";
import { Platform, StyleSheet, Switch, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { ActionSheet, Button, Card, ErrorState, ListRow, LoadingState, Screen, Text, TextField } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { authApi, profileApi } from "../../lib/api/endpoints";
import { useSession } from "../../lib/session/SessionProvider";
import { showToast } from "../../lib/toast";
import type { AppStackParamList } from "../../navigation/types";
import { useProfile, useProfileMutation } from "./hooks";

type Props = NativeStackScreenProps<AppStackParamList, "Settings">;

// react-native-web ignores thumbColor for the active state and needs its own prop.
const webThumb = Platform.OS === "web" ? { activeThumbColor: "#FFFFFF" } : {};

export default function SettingsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { signOut, forget, isModerator } = useSession();
  const { data: profile, isLoading, error, refetch } = useProfile();
  const update = useProfileMutation(profileApi.update);
  const [confirmSignOut, setConfirmSignOut] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");
  const [deleteBusy, setDeleteBusy] = useState(false);

  if (isLoading) return <LoadingState />;
  if (error || !profile) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  const toggle = (field: "discoverable" | "showDistance", value: boolean) =>
    update.mutate({ [field]: value }, { onError: (e) => showToast(errorMessage(e), "error") });

  const deleteAccount = async () => {
    setDeleteBusy(true);
    try {
      await authApi.deleteAccount(password);
      showToast("Votre compte et vos données ont été supprimés.", "success");
      await forget();
    } catch (e) {
      showToast(errorMessage(e), "error");
      setDeleteBusy(false);
    }
  };

  return (
    <Screen edges={["bottom"]} scroll>
      <Text variant="heading">Confidentialité</Text>
      <Card>
        <ListRow
          right={<Switch accessibilityLabel="Apparaître dans la découverte" onValueChange={(v) => toggle("discoverable", v)} {...webThumb} thumbColor="#fff" trackColor={{ true: colors.primary, false: colors.border }} value={profile.discoverable} />}
          subtitle="Désactivez pour mettre votre profil en pause. Vos matchs restent accessibles."
          title="Apparaître dans la découverte"
        />
        <ListRow
          right={<Switch accessibilityLabel="Afficher ma distance" onValueChange={(v) => toggle("showDistance", v)} {...webThumb} thumbColor="#fff" trackColor={{ true: colors.primary, false: colors.border }} value={profile.showDistance} />}
          subtitle="Une distance arrondie uniquement, jamais votre position."
          title="Afficher ma distance"
        />
        <ListRow icon="ban-outline" onPress={() => navigation.navigate("BlockedUsers")} title="Personnes bloquées" />
      </Card>

      {isModerator ? (
        <>
          <Text variant="heading">Modération</Text>
          <Card>
            <ListRow icon="flag-outline" onPress={() => navigation.navigate("Moderation")} subtitle="Signalements en attente" testID="settings-moderation" title="File de modération" />
          </Card>
        </>
      ) : null}

      <Text variant="heading">Compte</Text>
      <Card>
        <ListRow icon="log-out-outline" onPress={() => setConfirmSignOut(true)} testID="settings-signout" title="Se déconnecter" />
        <ListRow destructive icon="trash-outline" onPress={() => setDeleting(true)} testID="settings-delete" title="Supprimer mon compte" />
      </Card>

      {deleting ? (
        <View style={[styles.danger, { borderColor: colors.danger, backgroundColor: colors.dangerSoft }]}>
          <Text variant="label">Suppression définitive</Text>
          <Text tone="muted" variant="caption">
            Votre profil, vos photos, matchs, messages et notifications seront supprimés immédiatement. Cette action est irréversible.
          </Text>
          <TextField label="Confirmez avec votre mot de passe" onChangeText={setPassword} secureTextEntry testID="delete-password" value={password} />
          <View style={styles.row}>
            <Button label="Annuler" onPress={() => setDeleting(false)} size="sm" variant="ghost" />
            <Button disabled={!password} label="Supprimer définitivement" loading={deleteBusy} onPress={deleteAccount} size="sm" testID="delete-confirm" variant="danger" />
          </View>
        </View>
      ) : null}

      <ActionSheet
        actions={[{ label: "Se déconnecter", destructive: true, onPress: () => void signOut(), testID: "settings-signout-confirm" }]}
        onClose={() => setConfirmSignOut(false)}
        title="Se déconnecter ?"
        visible={confirmSignOut}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  danger: { borderWidth: 1, borderRadius: 16, padding: 16, gap: 10 },
  row: { flexDirection: "row", justifyContent: "flex-end", gap: 8 }
});
