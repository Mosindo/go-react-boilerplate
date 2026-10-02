import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { FlatList, StyleSheet, View } from "react-native";

import { ApiError } from "../../api/client";
import { authApi, profileApi, safetyApi } from "../../api/endpoints";
import { useAuth } from "../../auth/AuthContext";
import { useProfileStatus } from "../../hooks";
import { passwordError } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { keys } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { Button } from "../../ui/Button";
import { useFeedback } from "../../ui/Feedback";
import { Screen } from "../../ui/Screen";
import { EmptyState, ErrorState, Loading, errorMessage } from "../../ui/States";
import { SwitchRow } from "../../ui/SwitchRow";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";

type Props<K extends keyof MainStackParamList> = NativeStackScreenProps<MainStackParamList, K>;

export function SettingsScreen({ navigation }: Props<"Settings">) {
  const qc = useQueryClient();
  const { logout, user } = useAuth();
  const { toast, confirm } = useFeedback();
  const status = useProfileStatus();
  const profile = status.data?.profile;

  const save = useMutation({
    mutationFn: (patch: { discoverable?: boolean; showDistance?: boolean }) => {
      if (!profile) throw new Error("no profile");
      return profileApi.save({
        firstName: profile.firstName,
        birthDate: profile.birthDate,
        gender: profile.gender,
        bio: profile.bio,
        city: profile.city,
        latitude: profile.latitude,
        longitude: profile.longitude,
        interests: profile.interests.map((i) => i.slug),
        discoverable: patch.discoverable ?? profile.discoverable,
        showDistance: patch.showDistance ?? profile.showDistance,
      });
    },
    onSuccess: (data) => qc.setQueryData(keys.profile, data),
    onError: (err) => toast(errorMessage(err), "error"),
  });

  if (status.isLoading) return <Loading />;
  if (status.isError || !profile)
    return <ErrorState error={status.error} onRetry={() => void status.refetch()} />;

  return (
    <Screen scroll>
      <Text variant="heading">Confidentialité</Text>
      <SwitchRow
        label="Afficher mon profil dans la découverte"
        description="Désactivez pour faire une pause : vos matchs et conversations restent actifs."
        value={profile.discoverable}
        disabled={save.isPending}
        onValueChange={(v) => save.mutate({ discoverable: v })}
      />
      <SwitchRow
        label="Afficher ma distance"
        description="Les autres voient une distance approximative, jamais votre position."
        value={profile.showDistance}
        disabled={save.isPending}
        onValueChange={(v) => save.mutate({ showDistance: v })}
      />

      <Text variant="heading">Sécurité</Text>
      <Button
        label="Personnes bloquées"
        variant="secondary"
        onPress={() => navigation.navigate("Blocked")}
      />
      <Button
        label="Changer le mot de passe"
        variant="secondary"
        onPress={() => navigation.navigate("ChangePassword")}
      />

      <Text variant="heading">Compte</Text>
      <Text tone="muted">Connecté en tant que {user?.email}</Text>
      <Button
        label="Se déconnecter"
        variant="secondary"
        testID="logout"
        onPress={async () => {
          if (
            await confirm({
              title: "Se déconnecter ?",
              message: "Vous pourrez vous reconnecter à tout moment.",
              confirmLabel: "Se déconnecter",
            })
          ) {
            await logout();
          }
        }}
      />
      <Button
        label="Supprimer mon compte"
        variant="danger"
        onPress={() => navigation.navigate("DeleteAccount")}
      />
    </Screen>
  );
}

export function BlockedScreen() {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const query = useQuery({
    queryKey: keys.blocked,
    queryFn: async () => (await safetyApi.blocked()).blocks,
  });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.blocked });
      void qc.invalidateQueries({ queryKey: keys.discover });
      toast("Personne débloquée.", "success");
    },
    onError: (err) => toast(errorMessage(err), "error"),
  });

  if (query.isLoading) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  if (!query.data?.length)
    return (
      <EmptyState
        icon="shield-checkmark-outline"
        title="Personne n'est bloqué"
        message="Les personnes que vous bloquez apparaîtront ici."
      />
    );

  return (
    <FlatList
      data={query.data}
      keyExtractor={(b) => b.id}
      contentContainerStyle={styles.list}
      style={{ backgroundColor: colors.background }}
      renderItem={({ item }) => (
        <View
          style={[styles.blockRow, { borderColor: colors.border, backgroundColor: colors.surface }]}
        >
          <Text style={styles.flex}>{item.firstName || "Profil supprimé"}</Text>
          <Button label="Débloquer" variant="secondary" onPress={() => unblock.mutate(item.id)} />
        </View>
      )}
    />
  );
}

export function ChangePasswordScreen({ navigation }: Props<"ChangePassword">) {
  const { toast } = useFeedback();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      toast("Mot de passe modifié. Vos autres appareils sont déconnectés.", "success");
      navigation.goBack();
    },
    onError: (err) =>
      setError(
        err instanceof ApiError && err.code === "wrong_password"
          ? "Mot de passe actuel incorrect."
          : errorMessage(err),
      ),
  });

  const submit = () => {
    const problem = passwordError(next);
    if (!current || problem) {
      setError(problem ?? "Saisissez votre mot de passe actuel.");
      return;
    }
    setError(null);
    change.mutate();
  };

  return (
    <Screen scroll>
      <TextField
        label="Mot de passe actuel"
        value={current}
        onChangeText={setCurrent}
        secure
        autoComplete="current-password"
      />
      <TextField
        label="Nouveau mot de passe"
        value={next}
        onChangeText={setNext}
        secure
        autoComplete="new-password"
        hint="8 caractères minimum."
      />
      {error ? <Text tone="danger">{error}</Text> : null}
      <Button label="Modifier" onPress={submit} loading={change.isPending} />
    </Screen>
  );
}

export function DeleteAccountScreen() {
  const { forgetSession } = useAuth();
  const { confirm } = useFeedback();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: () => forgetSession(),
    onError: (err) =>
      setError(
        err instanceof ApiError && err.code === "wrong_password"
          ? "Mot de passe incorrect."
          : errorMessage(err),
      ),
  });

  const submit = async () => {
    if (!password) {
      setError("Saisissez votre mot de passe pour confirmer.");
      return;
    }
    const ok = await confirm({
      title: "Supprimer définitivement ?",
      message:
        "Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible.",
      confirmLabel: "Supprimer mon compte",
      destructive: true,
    });
    if (ok) remove.mutate();
  };

  return (
    <Screen scroll>
      <Text variant="title">Supprimer mon compte</Text>
      <Text tone="muted">
        Toutes vos données sont supprimées immédiatement : profil, photos, likes, matchs,
        conversations et signalements envoyés.
      </Text>
      <TextField
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secure
        error={error}
      />
      <Button
        label="Supprimer mon compte"
        variant="danger"
        onPress={() => void submit()}
        loading={remove.isPending}
        testID="delete-submit"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.sm },
  blockRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.md,
    borderWidth: 1,
  },
  flex: { flex: 1 },
});
