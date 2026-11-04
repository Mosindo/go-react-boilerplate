import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FlatList, View } from "react-native";
import { errorMessage } from "../../api/client";
import { authApi, profileApi, safetyApi } from "../../api/endpoints";
import { keys, useOwnProfile } from "../../hooks/queries";
import type { MainStackParams } from "../../navigation/types";
import { Screen } from "../../shared/layout";
import { Avatar, Button, EmptyState, ErrorState, Input, ListSkeleton, Loader, Text, spacing, useToast } from "../../shared/ui";
import { validatePassword } from "../../utils/validation";
import { AboutForm } from "../onboarding/AboutForm";
import { BasicsForm } from "../onboarding/BasicsForm";
import { PhotosManager } from "../onboarding/PhotosManager";
import { PreferencesForm } from "../onboarding/PreferencesForm";

type Props<K extends keyof MainStackParams> = NativeStackScreenProps<MainStackParams, K>;

export function EditProfileScreen({ navigation }: Props<"EditProfile">) {
  const qc = useQueryClient();
  const toast = useToast();
  const profile = useOwnProfile();
  const save = useMutation({
    mutationFn: profileApi.save,
    onSuccess: (p) => { qc.setQueryData(keys.profile, p); toast.show("Profil enregistré", "success"); navigation.goBack(); },
    onError: (err) => toast.show(errorMessage(err), "error"),
  });
  if (profile.isPending) return <Loader />;
  if (profile.isError) return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  const p = profile.data;
  return (
    <Screen scroll edges={["left", "right"]}>
      <Text variant="heading">Informations</Text>
      <BasicsForm profile={p} submitLabel="Enregistrer" busy={save.isPending} onSubmit={(v) => save.mutate({ firstName: v.firstName, gender: v.gender })} />
      <Text variant="heading">À propos de vous</Text>
      <AboutForm profile={p} showCity submitLabel="Enregistrer" busy={save.isPending} onSubmit={(v) => save.mutate({ firstName: p.firstName, gender: p.gender, bio: v.bio, city: v.city, interests: v.interests })} />
    </Screen>
  );
}

export function PhotosScreen() {
  const profile = useOwnProfile();
  if (profile.isPending) return <Loader />;
  if (profile.isError) return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  return (
    <Screen scroll edges={["left", "right"]}>
      <Text tone="muted">Touchez une photo pour la définir comme principale, la déplacer, la remplacer ou la supprimer. Gardez au moins une photo pour rester visible.</Text>
      <PhotosManager profile={profile.data} />
    </Screen>
  );
}

export function PreferencesScreen({ navigation }: Props<"Preferences">) {
  const qc = useQueryClient();
  const toast = useToast();
  const profile = useOwnProfile();
  const save = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (p) => { qc.setQueryData(keys.profile, p); toast.show("Préférences enregistrées", "success"); navigation.goBack(); },
    onError: (err) => toast.show(errorMessage(err), "error"),
  });
  if (profile.isPending) return <Loader />;
  if (profile.isError) return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  return (
    <Screen scroll edges={["left", "right"]}>
      <PreferencesForm initial={profile.data.preferences} submitLabel="Enregistrer" busy={save.isPending} onSubmit={(p) => save.mutate(p)} />
    </Screen>
  );
}

export function BlockedScreen() {
  const qc = useQueryClient();
  const toast = useToast();
  const query = useQuery({ queryKey: keys.blocked, queryFn: safetyApi.blocked });
  const unblock = useMutation({
    mutationFn: (id: string) => safetyApi.unblock(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.blocked });
      void qc.invalidateQueries({ queryKey: keys.conversations });
      toast.show("Personne débloquée", "success");
    },
    onError: (err) => toast.show(errorMessage(err), "error"),
  });
  if (query.isPending) return <ListSkeleton rows={3} />;
  if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  if (query.data.blocks.length === 0) return <EmptyState title="Aucune personne bloquée" />;
  return (
    <FlatList
      data={query.data.blocks}
      keyExtractor={(b) => b.userId}
      contentContainerStyle={{ padding: spacing.lg, gap: spacing.md }}
      renderItem={({ item }) => (
        <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
          <Avatar path={item.photoUrl} name={item.firstName || "?"} size={48} />
          <Text style={{ flex: 1 }} variant="label">{item.firstName || "Profil supprimé"}</Text>
          <Button title="Débloquer" variant="secondary" onPress={() => unblock.mutate(item.userId)} accessibilityLabel={`Débloquer ${item.firstName}`} />
        </View>
      )}
    />
  );
}

export function ChangePasswordScreen({ navigation }: Props<"ChangePassword">) {
  const toast = useToast();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => { toast.show("Mot de passe modifié. Vos autres appareils sont déconnectés.", "success"); navigation.goBack(); },
    onError: (err) => setError(errorMessage(err)),
  });
  const submit = () => {
    const problem = validatePassword(next);
    setError(problem);
    if (!problem) change.mutate();
  };
  return (
    <Screen scroll edges={["left", "right"]}>
      <Input label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureTextEntry />
      <Input label="Nouveau mot de passe" value={next} onChangeText={setNext} secureTextEntry hint="8 caractères minimum, avec une lettre et un chiffre" error={error} />
      <Button title="Modifier le mot de passe" onPress={submit} loading={change.isPending} disabled={!current || !next} />
    </Screen>
  );
}
