import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { Button, Header, Row, Screen, Section, Sheet, Text, TextField, showToast } from "../../ui";
import { passwordIssue } from "../../utils/validation";
import { confirm } from "../profile/ProfileDetailScreen";

export function AccountScreen() {
  const { user, signOut, deleteAccount } = useAuth();
  const [sheet, setSheet] = useState<"password" | "delete" | null>(null);
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const close = () => {
    setSheet(null);
    setCurrent("");
    setNext("");
    setError(null);
  };

  const changePassword = async () => {
    const issue = passwordIssue(next);
    if (issue) return setError(issue);
    setLoading(true);
    try {
      await authApi.changePassword(current, next);
      showToast("Mot de passe modifié. Vos autres sessions ont été déconnectées.", "success");
      close();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const remove = async () => {
    if (
      !(await confirm(
        "Supprimer définitivement votre compte ?",
        "Profil, photos, matchs et messages seront effacés. Cette action est irréversible.",
        "Supprimer"
      ))
    )
      return;
    setLoading(true);
    try {
      await deleteAccount(current);
      showToast("Votre compte et vos données ont été supprimés.", "success");
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll>
      <Header back title="Compte" />
      <Section title="Connexion">
        <Row icon="mail-outline" label="Email" last value={user?.email} />
      </Section>
      <Section>
        <Row icon="key-outline" label="Changer de mot de passe" onPress={() => setSheet("password")} />
        <Row icon="log-out-outline" label="Se déconnecter" last onPress={signOut} testID="account-logout" />
      </Section>
      <Section footer="La suppression efface immédiatement votre profil, vos photos, vos matchs et vos messages." title="Zone sensible">
        <Row icon="trash-outline" label="Supprimer mon compte" last onPress={() => setSheet("delete")} testID="account-delete" tone="danger" />
      </Section>

      <Sheet onClose={close} title={sheet === "delete" ? "Supprimer mon compte" : "Changer de mot de passe"} visible={!!sheet}>
        <View style={styles.body}>
          {sheet === "delete" ? <Text tone="muted">Confirmez avec votre mot de passe. Cette action est définitive.</Text> : null}
          <TextField
            autoComplete="current-password"
            label={sheet === "delete" ? "Mot de passe" : "Mot de passe actuel"}
            onChangeText={setCurrent}
            secureTextEntry
            testID="account-current-password"
            value={current}
          />
          {sheet === "password" ? (
            <TextField
              autoComplete="new-password"
              hint="8 caractères minimum, avec une lettre et un chiffre."
              label="Nouveau mot de passe"
              onChangeText={setNext}
              secureTextEntry
              value={next}
            />
          ) : null}
          {error ? (
            <Text tone="danger" variant="small">
              {error}
            </Text>
          ) : null}
          {sheet === "delete" ? (
            <Button disabled={!current} label="Supprimer définitivement" loading={loading} onPress={remove} testID="account-delete-confirm" variant="danger" />
          ) : (
            <Button disabled={!current || !next} label="Changer" loading={loading} onPress={changePassword} />
          )}
        </View>
      </Sheet>
    </Screen>
  );
}

const styles = StyleSheet.create({ body: { gap: 14, paddingBottom: 8 } });
