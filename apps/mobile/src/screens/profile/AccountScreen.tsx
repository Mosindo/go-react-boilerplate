import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useAuth } from "../../auth/AuthProvider";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Header } from "../../components/Header";
import { Row, Section } from "../../components/Section";
import { Screen } from "../../components/Screen";
import { Sheet } from "../../components/Sheet";
import { TextField } from "../../components/TextField";
import { colors, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";

export default function AccountScreen() {
  const navigation = useNavigation();
  const { account, signOut, deleteAccount } = useAuth();
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const confirmDelete = async () => {
    if (!password) {
      setError("Confirme avec ton mot de passe.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await deleteAccount(password);
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Header title="Compte" onBack={() => navigation.goBack()} />
      <View style={styles.body}>
        <Section title="Connexion">
          <Row icon="mail-outline" label="E-mail" value={account?.email} />
          <Row icon="log-out-outline" label="Se déconnecter" onPress={() => void signOut()} testID="account-signout" />
        </Section>
        <Section title="Zone sensible" footer="La suppression est immédiate et définitive : profil, photos, matchs, messages et notifications sont effacés.">
          <Row icon="trash-outline" label="Supprimer mon compte" destructive onPress={() => setDeleting(true)} testID="account-delete" />
        </Section>
      </View>
      <Sheet visible={deleting} title="Supprimer mon compte" onClose={() => setDeleting(false)}>
        <AppText color={colors.textMuted}>Cette action est irréversible. Toutes tes données seront supprimées de nos serveurs.</AppText>
        <TextField label="Mot de passe" value={password} onChangeText={setPassword} secureToggle autoComplete="current-password" testID="account-delete-password" />
        {error ? <AppText color={colors.danger}>{error}</AppText> : null}
        <Button label="Supprimer définitivement" variant="danger" onPress={confirmDelete} loading={busy} testID="account-delete-confirm" />
      </Sheet>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { gap: spacing.lg, marginTop: spacing.sm }
});
