import React, { useState } from "react";
import { KeyboardAvoidingView, Modal, Platform, StyleSheet, View } from "react-native";
import { deleteAccount } from "../api/auth";
import { friendlyError } from "../api/errors";
import { Button, Input, Notice, Text, colors, radii, spacing } from "../shared/ui";

type DeleteAccountSheetProps = {
  visible: boolean;
  onClose: () => void;
  onDeleted: () => void;
};

/** Irreversible: asks for the password again, as the server does. */
export function DeleteAccountSheet({ visible, onClose, onDeleted }: DeleteAccountSheetProps) {
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const close = () => {
    setPassword("");
    setError(null);
    onClose();
  };

  const confirm = async () => {
    setBusy(true);
    setError(null);
    try {
      await deleteAccount(password);
      setPassword("");
      onDeleted();
    } catch (err) {
      setError(friendlyError(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal animationType="slide" onRequestClose={close} transparent visible={visible}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.backdrop}>
        <View style={styles.sheet}>
          <Text variant="heading" weight="bold">
            Supprimer mon compte
          </Text>
          <Text tone="muted">
            Votre profil, vos photos, vos matchs et tous vos messages seront définitivement supprimés. Cette action est irréversible.
          </Text>
          <Input
            autoCapitalize="none"
            label="Confirmez avec votre mot de passe"
            onChangeText={setPassword}
            secureTextEntry
            testID="delete-password"
            textContentType="password"
            value={password}
          />
          {error ? <Notice title={error} tone="danger" /> : null}
          <View style={styles.buttons}>
            <Button label="Annuler" onPress={close} style={styles.flex} variant="outline" />
            <Button
              disabled={!password}
              label="Supprimer"
              loading={busy}
              onPress={() => void confirm()}
              style={styles.flex}
              testID="delete-confirm"
              variant="destructive"
            />
          </View>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, justifyContent: "flex-end" },
  sheet: {
    backgroundColor: colors.backgroundElevated,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md
  },
  buttons: { flexDirection: "row", gap: spacing.md },
  flex: { flex: 1 }
});
