import React, { useState } from "react";
import { Alert, Modal, Pressable, StyleSheet, View } from "react-native";
import { useAuth } from "../auth/AuthContext";
import { useMyProfile, useUnreadNotifications } from "../features/hooks";
import { describeError } from "../lib/validation";
import { useNav } from "../nav/Navigation";
import { radii, useTheme } from "../theme/theme";
import { Button, Input, PhotoImage, Screen, Text, useToast } from "../ui";

function Row({ label, hint, onPress }: { label: string; hint?: string; onPress: () => void }) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={[styles.row, { backgroundColor: colors.surface, borderColor: colors.border }]}
    >
      <Text variant="label" style={styles.flex}>
        {label}
      </Text>
      {hint ? (
        <Text variant="caption" muted>
          {hint}
        </Text>
      ) : null}
      <Text muted>›</Text>
    </Pressable>
  );
}

export default function MeScreen() {
  const { colors } = useTheme();
  const nav = useNav();
  const toast = useToast();
  const { logout, deleteAccount, user } = useAuth();
  const me = useMyProfile();
  const unread = useUnreadNotifications();
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const profile = me.data;

  const confirmDelete = async () => {
    if (!password) return;
    setBusy(true);
    try {
      await deleteAccount(password);
    } catch (error) {
      toast.show(describeError(error), "error");
      setBusy(false);
    }
  };

  return (
    <Screen title="Mon profil" scroll noBottomInset>
      {profile ? (
        <View style={styles.hero}>
          <PhotoImage url={profile.photos[0]?.url} name={profile.firstName} circle style={styles.avatar} />
          <Text variant="heading">
            {profile.firstName}, {profile.age}
          </Text>
          <Text variant="caption" muted>
            {user?.email}
          </Text>
        </View>
      ) : null}
      <View style={styles.group}>
        <Row label="Modifier mon profil" onPress={() => nav.push({ name: "editProfile" })} />
        <Row
          label="Mes photos"
          hint={profile ? `${profile.photos.length}/6` : undefined}
          onPress={() => nav.push({ name: "editPhotos" })}
        />
        <Row label="Préférences de rencontre" onPress={() => nav.push({ name: "editPreferences" })} />
        <Row label="Confidentialité" onPress={() => nav.push({ name: "privacy" })} />
        <Row
          label="Notifications"
          hint={unread.data ? `${unread.data} non lues` : undefined}
          onPress={() => nav.push({ name: "notifications" })}
        />
        <Row label="Personnes bloquées" onPress={() => nav.push({ name: "blocked" })} />
      </View>
      <Button title="Se déconnecter" variant="secondary" onPress={() => void logout()} testID="logout" />
      <Button
        title="Supprimer mon compte"
        variant="ghost"
        onPress={() =>
          Alert.alert(
            "Supprimer mon compte ?",
            "Votre profil, vos photos, vos matchs et vos messages seront définitivement supprimés. Cette action est irréversible.",
            [
              { text: "Annuler", style: "cancel" },
              { text: "Continuer", style: "destructive", onPress: () => setDeleting(true) }
            ]
          )
        }
      />
      <Modal visible={deleting} transparent animationType="fade" onRequestClose={() => setDeleting(false)}>
        <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
          <View style={[styles.dialog, { backgroundColor: colors.background }]}>
            <Text variant="heading">Confirmez votre mot de passe</Text>
            <Input
              label="Mot de passe"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              autoComplete="current-password"
              testID="delete-password"
            />
            <Button
              title="Supprimer définitivement"
              variant="danger"
              onPress={confirmDelete}
              loading={busy}
              disabled={!password}
              testID="delete-confirm"
            />
            <Button
              title="Annuler"
              variant="ghost"
              onPress={() => {
                setDeleting(false);
                setPassword("");
              }}
            />
          </View>
        </View>
      </Modal>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  hero: { alignItems: "center", gap: 6 },
  avatar: { width: 110, height: 110, marginBottom: 6 },
  group: { gap: 10 },
  row: { flexDirection: "row", alignItems: "center", gap: 8, padding: 16, borderRadius: radii.md, borderWidth: 1 },
  backdrop: { flex: 1, justifyContent: "center", padding: 24 },
  dialog: { borderRadius: radii.lg, padding: 20, gap: 14 }
});
