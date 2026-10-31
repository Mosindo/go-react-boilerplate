import React, { useState } from "react";
import { Alert, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as authApi from "../api/auth";
import { ApiError } from "../api/client";
import { getBlocked, saveProfile, unblockUser } from "../api/dating";
import type { OwnProfile } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { Chip } from "../components/Chip";
import { toInput } from "../components/ProfileForm";
import { useOwnProfile } from "../hooks/useOwnProfile";
import { queryKeys } from "../lib/queryKeys";
import { Screen } from "../shared/layout";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Button, Input, Notice, Text, colors, radii, spacing } from "../shared/ui";

function Toggle({
  title,
  description,
  value,
  onChange
}: {
  title: string;
  description: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <View style={styles.toggle}>
      <View style={styles.flex}>
        <Text weight="semibold">{title}</Text>
        <Text tone="muted" variant="caption">
          {description}
        </Text>
      </View>
      <Chip label={value ? "Activé" : "Désactivé"} onPress={() => onChange(!value)} selected={value} />
    </View>
  );
}

function Content({ profile }: { profile: OwnProfile }) {
  const queryClient = useQueryClient();
  const { dropSession } = useAuth();
  const [password, setPassword] = useState("");
  const [confirming, setConfirming] = useState(false);

  const blocked = useQuery({ queryKey: queryKeys.blocked, queryFn: getBlocked });

  const update = useMutation({
    mutationFn: (patch: Partial<Pick<OwnProfile, "isDiscoverable" | "showDistance">>) => saveProfile({ ...toInput(profile), ...patch }),
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.profile, saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.feed });
    }
  });
  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
      void queryClient.invalidateQueries({ queryKey: queryKeys.feed });
    }
  });
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: () => dropSession()
  });

  const deleteError = remove.error
    ? remove.error instanceof ApiError && remove.error.status === 403
      ? "Mot de passe incorrect."
      : "Suppression impossible. Réessayez."
    : null;

  return (
    <Screen scroll>
      <Text variant="heading" weight="bold">
        Visibilité
      </Text>
      <Toggle
        description="Désactivé : vous n'apparaissez plus dans la découverte (vos matchs restent actifs)."
        onChange={(v) => update.mutate({ isDiscoverable: v })}
        title="Apparaître dans la découverte"
        value={profile.isDiscoverable}
      />
      <Toggle
        description="Désactivé : les autres ne voient pas votre distance approximative."
        onChange={(v) => update.mutate({ showDistance: v })}
        title="Afficher ma distance"
        value={profile.showDistance}
      />

      <Text variant="heading" weight="bold">
        Personnes bloquées
      </Text>
      {blocked.isLoading ? <LoadingView /> : null}
      {blocked.isError ? <Notice title="Liste indisponible." tone="warning" /> : null}
      {blocked.data?.length === 0 ? <Text tone="muted">Vous n'avez bloqué personne.</Text> : null}
      {blocked.data?.map((b) => (
        <View key={b.id} style={styles.toggle}>
          <Text style={styles.flex} weight="semibold">
            {b.firstName || "Utilisateur"}
          </Text>
          <Button label="Débloquer" loading={unblock.isPending} onPress={() => unblock.mutate(b.id)} size="sm" variant="outline" />
        </View>
      ))}

      <Text variant="heading" weight="bold">
        Supprimer mon compte
      </Text>
      <Text tone="muted">
        Votre profil, vos photos, vos likes, vos matchs et vos messages sont effacés définitivement. Cette action est irréversible.
      </Text>
      {confirming ? (
        <View style={styles.confirm}>
          <Input
            autoComplete="password"
            label="Confirmez avec votre mot de passe"
            onChangeText={setPassword}
            secureTextEntry
            testID="delete-password"
            value={password}
          />
          {deleteError ? <Notice title={deleteError} tone="danger" /> : null}
          <Button
            disabled={!password}
            label="Supprimer définitivement"
            loading={remove.isPending}
            onPress={() =>
              Alert.alert("Supprimer votre compte ?", "Cette action est définitive.", [
                { text: "Annuler", style: "cancel" },
                { text: "Supprimer", style: "destructive", onPress: () => remove.mutate() }
              ])
            }
            testID="delete-confirm"
            variant="destructive"
          />
          <Button
            label="Annuler"
            onPress={() => {
              setConfirming(false);
              setPassword("");
            }}
            variant="ghost"
          />
        </View>
      ) : (
        <Button label="Supprimer mon compte" onPress={() => setConfirming(true)} testID="delete-account" variant="outline" />
      )}
    </Screen>
  );
}

export default function PrivacyScreen() {
  const { data, isLoading, isError, refetch } = useOwnProfile();
  if (isLoading)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );
  if (isError || !data)
    return (
      <Screen>
        <ErrorView actionLabel="Réessayer" message="Paramètres indisponibles." onAction={() => void refetch()} />
      </Screen>
    );
  return <Content profile={data} />;
}

const styles = StyleSheet.create({
  flex: { flex: 1, gap: 2 },
  toggle: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  confirm: { gap: spacing.md }
});
