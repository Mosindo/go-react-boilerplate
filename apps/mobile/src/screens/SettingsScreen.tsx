import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { changePassword } from "../api/auth";
import { savePrivacy } from "../api/platform";
import { useAuth } from "../hooks/useAuth";
import { useMyProfile } from "../hooks/queries";
import { confirm } from "../lib/confirm";
import { markDeckStale } from "../lib/deckSignal";
import { errorMessage } from "../lib/errors";
import { passwordProblem } from "../lib/validation";
import { queryKeys } from "../realtime/cache";
import { ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { Header, ScreenContainer, Section } from "../shared/layout";
import { Button, Card, Input, Notice, SwitchRow, Text } from "../shared/ui";

export default function SettingsScreen() {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const { user, logout, deleteAccount } = useAuth();
  const { data, isLoading, isError, error, refetch } = useMyProfile();
  const [savingPrivacy, setSavingPrivacy] = useState(false);
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [pwError, setPwError] = useState<string | null>(null);
  const [pwSaving, setPwSaving] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  if (isLoading) return <LoadingView />;
  if (isError || !data?.profile) return <ErrorView error={error} onRetry={() => void refetch()} />;
  const profile = data.profile;

  const updatePrivacy = async (isDiscoverable: boolean, showDistance: boolean) => {
    setSavingPrivacy(true);
    try {
      await savePrivacy(isDiscoverable, showDistance);
      await queryClient.invalidateQueries({ queryKey: queryKeys.me });
      markDeckStale();
    } catch (e) {
      showError(errorMessage(e));
    } finally {
      setSavingPrivacy(false);
    }
  };

  const submitPassword = async () => {
    const problem = passwordProblem(next);
    if (!current || problem) {
      setPwError(problem ?? "Entrez votre mot de passe actuel.");
      return;
    }
    setPwSaving(true);
    setPwError(null);
    try {
      await changePassword(current, next);
      setCurrent("");
      setNext("");
      showSuccess("Mot de passe modifié. Vos autres appareils ont été déconnectés.");
    } catch (e) {
      setPwError(errorMessage(e));
    } finally {
      setPwSaving(false);
    }
  };

  const submitDelete = async () => {
    if (!deletePassword) {
      setDeleteError("Entrez votre mot de passe pour confirmer.");
      return;
    }
    const ok = await confirm("Supprimer définitivement votre compte ?", "Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible.", "Supprimer mon compte", true);
    if (!ok) return;
    setDeleting(true);
    setDeleteError(null);
    try {
      await deleteAccount(deletePassword);
    } catch (e) {
      setDeleteError(errorMessage(e));
      setDeleting(false);
    }
  };

  return (
    <ScreenContainer scroll testID="settings-screen">
      <Header title="Réglages" onBack={() => navigation.goBack()} />

      <Section title="Confidentialité">
        <SwitchRow
          label="Profil visible"
          description="Désactivez pour mettre votre profil en pause : plus personne ne vous découvre, vos matchs restent actifs."
          value={profile.isDiscoverable}
          disabled={savingPrivacy}
          onValueChange={(value) => void updatePrivacy(value, profile.showDistance)}
        />
        <SwitchRow
          label="Afficher ma distance et ma ville"
          description="Les autres voient une distance approximative, jamais votre position exacte."
          value={profile.showDistance}
          disabled={savingPrivacy}
          onValueChange={(value) => void updatePrivacy(profile.isDiscoverable, value)}
        />
      </Section>

      <Section title="Compte" description={user?.email}>
        <Input testID="pw-current" label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureTextEntry autoCapitalize="none" />
        <Input testID="pw-new" label="Nouveau mot de passe" value={next} onChangeText={setNext} secureTextEntry autoCapitalize="none" hint="8 caractères minimum." />
        {pwError ? <Notice kind="error" message={pwError} /> : null}
        <Button testID="pw-save" label="Changer le mot de passe" variant="secondary" onPress={submitPassword} loading={pwSaving} />
        <Button testID="logout" label="Se déconnecter" variant="secondary" onPress={() => void logout()} />
      </Section>

      <Section title="Zone sensible">
        <Card>
          <Text variant="bodyStrong">Supprimer mon compte</Text>
          <Text variant="caption" tone="muted">
            Efface définitivement votre profil, vos photos, vos matchs et vos conversations.
          </Text>
          <Input testID="delete-password" label="Mot de passe" value={deletePassword} onChangeText={setDeletePassword} secureTextEntry autoCapitalize="none" />
          {deleteError ? <Notice kind="error" message={deleteError} /> : null}
          <Button testID="delete-account" label="Supprimer mon compte" variant="danger" onPress={submitDelete} loading={deleting} />
        </Card>
      </Section>
    </ScreenContainer>
  );
}
