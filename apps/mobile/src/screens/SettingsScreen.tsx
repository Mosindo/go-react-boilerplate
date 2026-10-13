import React, { useState } from "react";
import { ScrollView, StyleSheet } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { unblockUser, saveProfile } from "../api/platform";
import { errorMessage } from "../api/client";
import { useBlocked, useOwnProfile } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { queryKeys } from "../lib/queryClient";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer, Section } from "../shared/layout";
import { Button, Card, ConfirmDialog, Input, ListItem, Notice, Text, Toggle, spacing } from "../shared/ui";

export default function SettingsScreen() {
  const queryClient = useQueryClient();
  const { signOut, deleteAccount, user } = useAuth();
  const profileQuery = useOwnProfile();
  const blocked = useBlocked();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [password, setPassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const profile = profileQuery.data;

  const privacy = useMutation({
    mutationFn: (patch: { isVisible?: boolean; showDistance?: boolean }) => {
      if (!profile) {
        return Promise.reject(new Error("Profil indisponible"));
      }
      return saveProfile({
        firstName: profile.firstName,
        birthDate: profile.birthDate,
        gender: profile.gender,
        bio: profile.bio,
        city: profile.city,
        interestIds: profile.interestIds,
        isVisible: patch.isVisible ?? profile.isVisible,
        showDistance: patch.showDistance ?? profile.showDistance
      });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile }),
    onError: (e) => setError(errorMessage(e))
  });

  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.blocked }),
    onError: (e) => setError(errorMessage(e))
  });

  const remove = useMutation({
    mutationFn: () => deleteAccount(password),
    onError: (e) => setDeleteError(errorMessage(e))
  });

  if (profileQuery.isLoading) {
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  }
  if (!profile) {
    return (
      <ScreenContainer>
        <ErrorView message="Impossible de charger vos réglages." onAction={() => void profileQuery.refetch()} />
      </ScreenContainer>
    );
  }

  return (
    <ScreenContainer>
      <ScrollView contentContainerStyle={styles.scroll}>
        {error ? <Notice message={error} tone="danger" /> : null}

        <Section title="Confidentialité">
          <Card>
            <Toggle
              description="Désactivé, votre profil n'est plus montré aux autres (vos matchs gardent l'accès)."
              disabled={privacy.isPending}
              label="Apparaître dans la découverte"
              onChange={(isVisible) => privacy.mutate({ isVisible })}
              value={profile.isVisible}
            />
            <Toggle
              description="Affiche une distance approximative (par tranches de 5 km). Jamais votre position exacte."
              disabled={privacy.isPending}
              label="Afficher ma distance"
              onChange={(showDistance) => privacy.mutate({ showDistance })}
              value={profile.showDistance}
            />
          </Card>
        </Section>

        <Section description="Les personnes bloquées ne peuvent ni vous voir ni vous écrire." title="Personnes bloquées">
          <Card padded={false}>
            {blocked.isLoading ? (
              <Text style={styles.pad} tone="muted">
                Chargement…
              </Text>
            ) : (blocked.data ?? []).length === 0 ? (
              <Text style={styles.pad} tone="muted">
                Vous n'avez bloqué personne.
              </Text>
            ) : (
              (blocked.data ?? []).map((b) => (
                <ListItem
                  key={b.userId}
                  title={b.firstName || "Ancien membre"}
                  trailing={<Button label="Débloquer" onPress={() => unblock.mutate(b.userId)} size="sm" variant="outline" />}
                />
              ))
            )}
          </Card>
        </Section>

        <Section title="Compte">
          <Card>
            <Text tone="muted" variant="label">
              Connecté en tant que {user?.email}
            </Text>
            <Button fullWidth label="Se déconnecter" onPress={() => void signOut()} testID="settings-logout" variant="outline" />
            <Button
              fullWidth
              label="Supprimer mon compte"
              onPress={() => setConfirmDelete(true)}
              testID="settings-delete"
              variant="destructive"
            />
          </Card>
        </Section>
      </ScrollView>

      <ConfirmDialog
        confirmLabel="Supprimer définitivement"
        destructive
        loading={remove.isPending}
        message="Votre profil, vos photos, vos matchs et vos messages seront effacés immédiatement. Cette action est irréversible."
        onCancel={() => {
          setConfirmDelete(false);
          setPassword("");
          setDeleteError(null);
        }}
        onConfirm={() => {
          setDeleteError(null);
          remove.mutate();
        }}
        title="Supprimer mon compte ?"
        visible={confirmDelete}
      >
        <Input
          autoCapitalize="none"
          label="Confirmez avec votre mot de passe"
          onChangeText={setPassword}
          secureTextEntry
          testID="delete-password"
          value={password}
        />
        {deleteError ? <Notice message={deleteError} tone="danger" /> : null}
      </ConfirmDialog>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  scroll: { gap: spacing.xl, paddingBottom: spacing.xxxl },
  pad: { padding: spacing.lg }
});
