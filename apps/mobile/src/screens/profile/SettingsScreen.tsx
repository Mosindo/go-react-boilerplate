import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useClearLocation, useMyProfile, useUpdateLocation, useUpdateSettings } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { ConfirmDialog, LoadingView, showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Card, ListRow, Text, ToggleRow, spacing } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";
import { readApproximateLocation } from "../../utils/media";

type Props = NativeStackScreenProps<AppStackParamList, "Settings">;

export default function SettingsScreen({ navigation }: Props) {
  const { signOut } = useAuth();
  const profile = useMyProfile();
  const settings = useUpdateSettings();
  const updateLocation = useUpdateLocation();
  const clearLocation = useClearLocation();
  const [confirmClear, setConfirmClear] = useState(false);
  const [locating, setLocating] = useState(false);

  if (!profile.data) {
    return <LoadingView />;
  }
  const p = profile.data;

  const toggle = (patch: Partial<typeof p.settings>) =>
    settings.mutate({ ...p.settings, ...patch }, { onError: (e) => showToast(errorMessage(e), "danger") });

  const relocate = async () => {
    setLocating(true);
    const result = await readApproximateLocation();
    setLocating(false);
    if ("error" in result) {
      showToast(result.error, "danger");
      return;
    }
    updateLocation.mutate(result, { onSuccess: () => showToast("Position mise à jour", "success") });
  };

  return (
    <Screen testID="settings-screen">
      <ScreenHeader title="Réglages" />

      <Text tone="muted" variant="eyebrow">
        Confidentialité
      </Text>
      <Card>
        <ToggleRow
          description="Désactive pour mettre ton profil en pause. Tes matchs restent accessibles."
          onValueChange={(v) => toggle({ isDiscoverable: v })}
          testID="settings-discoverable"
          title="Apparaître dans Découvrir"
          value={p.settings.isDiscoverable}
        />
        <ToggleRow
          description="Les autres ne verront pas à combien de kilomètres tu te trouves."
          onValueChange={(v) => toggle({ hideDistance: v })}
          title="Masquer ma distance"
          value={p.settings.hideDistance}
        />
      </Card>

      <Text tone="muted" variant="eyebrow">
        Localisation
      </Text>
      <Card>
        <Text tone="muted" variant="label">
          {p.hasLocation
            ? `Position approximative enregistrée${p.city ? ` (${p.city})` : ""}. Elle est arrondie à ~1 km et n'est jamais partagée.`
            : "Aucune position enregistrée : tu n'apparais pas dans Découvrir."}
        </Text>
        <Button
          label={p.hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"}
          loading={locating || updateLocation.isPending}
          onPress={() => void relocate()}
          size="sm"
          variant="secondary"
        />
        {p.hasLocation ? <Button label="Effacer ma position" onPress={() => setConfirmClear(true)} size="sm" variant="ghost" /> : null}
      </Card>

      <Text tone="muted" variant="eyebrow">
        Sécurité
      </Text>
      <Card padded={false} style={{ paddingHorizontal: spacing.lg }}>
        <ListRow onPress={() => navigation.navigate("BlockedUsers")} title="Personnes bloquées" />
        <ListRow onPress={() => navigation.navigate("ChangePassword")} title="Changer de mot de passe" />
      </Card>

      <Text tone="muted" variant="eyebrow">
        Compte
      </Text>
      <Card padded={false} style={{ paddingHorizontal: spacing.lg }}>
        <ListRow onPress={() => void signOut()} testID="settings-logout" title="Se déconnecter" />
        <ListRow onPress={() => navigation.navigate("DeleteAccount")} testID="settings-delete" title="Supprimer mon compte" tone="danger" />
      </Card>

      <ConfirmDialog
        confirmLabel="Effacer"
        destructive
        message="Tu n'apparaîtras plus dans Découvrir tant que tu n'auras pas renseigné une nouvelle position."
        onCancel={() => setConfirmClear(false)}
        onConfirm={() => {
          setConfirmClear(false);
          clearLocation.mutate(undefined, { onSuccess: () => showToast("Position effacée") });
        }}
        title="Effacer ta position ?"
        visible={confirmClear}
      />
    </Screen>
  );
}
