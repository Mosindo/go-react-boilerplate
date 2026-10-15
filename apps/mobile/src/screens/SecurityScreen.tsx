import React, { useState } from "react";
import { View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { changePassword, deleteAccount } from "../api/auth";
import { useAuth } from "../hooks/useAuth";
import { errorMessage } from "../i18n/errors";
import { checkPassword } from "../lib/validation";
import { confirm } from "../lib/confirm";
import type { RootStackParamList } from "../navigation/types";
import { showSuccess } from "../shared/feedback";
import { Header, ScreenContainer, Section } from "../shared/layout";
import { Button, FormField, Notice, Text, spacing } from "../shared/ui";

export default function SecurityScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { accountDeleted, signOut } = useAuth();

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [changing, setChanging] = useState(false);
  const [changeError, setChangeError] = useState<string | null>(null);

  const [deletePassword, setDeletePassword] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const submitChange = async () => {
    setChangeError(null);
    if (!current) {
      setChangeError("Saisissez votre mot de passe actuel.");
      return;
    }
    if (checkPassword(next)) {
      setChangeError("Le nouveau mot de passe doit contenir entre 8 et 72 caractères.");
      return;
    }
    setChanging(true);
    try {
      await changePassword(current, next);
      showSuccess("Mot de passe modifié. Reconnectez-vous.");
      await signOut(); // The server revoked every session, this device included.
    } catch (error) {
      setChangeError(errorMessage(error));
      setChanging(false);
    }
  };

  const runDelete = async () => {
    setDeleting(true);
    try {
      await deleteAccount(deletePassword);
      await accountDeleted();
      showSuccess("Votre compte a été supprimé.");
    } catch (error) {
      setDeleteError(errorMessage(error));
      setDeleting(false);
    }
  };

  const submitDelete = () => {
    setDeleteError(null);
    if (!deletePassword) {
      setDeleteError("Saisissez votre mot de passe pour confirmer.");
      return;
    }
    confirm({
      title: "Supprimer définitivement votre compte ?",
      message: "Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible.",
      confirmLabel: "Supprimer",
      destructive: true,
      onConfirm: () => void runDelete()
    });
  };

  return (
    <ScreenContainer keyboardAware scroll testID="security-screen">
      <Header onBack={() => navigation.goBack()} title="Mot de passe et compte" />

      <Section title="CHANGER MON MOT DE PASSE">
        <FormField
          label="Mot de passe actuel"
          onChangeText={setCurrent}
          secureTextEntry
          testID="current-password"
          textContentType="password"
          value={current}
        />
        <FormField
          hint="Au moins 8 caractères."
          label="Nouveau mot de passe"
          onChangeText={setNext}
          secureTextEntry
          testID="new-password"
          textContentType="newPassword"
          value={next}
        />
        {changeError ? <Notice message={changeError} tone="danger" /> : null}
        <Button fullWidth label="Modifier le mot de passe" loading={changing} onPress={() => void submitChange()} variant="secondary" />
        <Text tone="subtle" variant="caption">
          Vous serez déconnecté de tous vos appareils.
        </Text>
      </Section>

      <Section title="SUPPRIMER MON COMPTE">
        <View style={{ gap: spacing.md }}>
          <Notice
            message="Toutes vos données sont supprimées : profil, photos, likes, matchs et conversations. Les personnes avec qui vous aviez matché perdent la conversation."
            title="Action irréversible"
            tone="warning"
          />
          <FormField
            label="Mot de passe"
            onChangeText={setDeletePassword}
            secureTextEntry
            testID="delete-password"
            value={deletePassword}
          />
          {deleteError ? <Notice message={deleteError} tone="danger" /> : null}
          <Button
            fullWidth
            label="Supprimer mon compte"
            loading={deleting}
            onPress={submitDelete}
            testID="delete-account"
            variant="destructive"
          />
        </View>
      </Section>
    </ScreenContainer>
  );
}
