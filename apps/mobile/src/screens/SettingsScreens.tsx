import React from "react";
import { FlatList, StyleSheet, Switch, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi, safetyApi } from "../api/endpoints";
import type { OwnProfile, Preferences, ProfileInput } from "../api/types";
import { PhotosEditor } from "../features/PhotosEditor";
import { PreferencesForm } from "../features/PreferencesForm";
import { ProfileForm } from "../features/ProfileForm";
import { useMyProfile } from "../features/hooks";
import { keys } from "../features/keys";
import { describeError } from "../lib/validation";
import { useNav } from "../nav/Navigation";
import { useTheme } from "../theme/theme";
import { Button, EmptyState, ErrorState, LoadingView, Screen, Text, useToast } from "../ui";

function useProfileSaver() {
  const queryClient = useQueryClient();
  const toast = useToast();
  const nav = useNav();
  return useMutation({
    mutationFn: (input: ProfileInput) => profileApi.save(input),
    onSuccess: () => {
      toast.show("Profil mis à jour.");
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      nav.back();
    },
    onError: (e) => toast.show(describeError(e), "error")
  });
}

function WithProfile({ title, children }: { title: string; children: (profile: OwnProfile) => React.ReactNode }) {
  const nav = useNav();
  const me = useMyProfile();
  return (
    <Screen title={title} onBack={nav.back} scroll>
      {me.isPending ? <LoadingView /> : null}
      {me.isError ? <ErrorState message={describeError(me.error)} onRetry={() => void me.refetch()} /> : null}
      {me.data ? children(me.data) : null}
    </Screen>
  );
}

export function EditProfileScreen() {
  const save = useProfileSaver();
  return (
    <WithProfile title="Modifier mon profil">
      {(profile) => <ProfileForm initial={profile} submitLabel="Enregistrer" busy={save.isPending} onSubmit={(i) => save.mutate(i)} />}
    </WithProfile>
  );
}

export function EditPhotosScreen() {
  const nav = useNav();
  return (
    <Screen title="Mes photos" onBack={nav.back} scroll>
      <PhotosEditor />
    </Screen>
  );
}

export function EditPreferencesScreen() {
  const nav = useNav();
  const toast = useToast();
  const queryClient = useQueryClient();
  const save = useMutation({
    mutationFn: (p: Preferences) => profileApi.savePreferences(p),
    onSuccess: () => {
      toast.show("Préférences enregistrées.");
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      nav.back();
    },
    onError: (e) => toast.show(describeError(e), "error")
  });
  return (
    <WithProfile title="Préférences de rencontre">
      {(profile) => (
        <PreferencesForm initial={profile.preferences} submitLabel="Enregistrer" busy={save.isPending} onSubmit={(p) => save.mutate(p)} />
      )}
    </WithProfile>
  );
}

function ToggleRow({ label, hint, value, onChange }: { label: string; hint: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <View style={styles.toggle}>
      <View style={styles.flex}>
        <Text variant="label">{label}</Text>
        <Text variant="caption" muted>
          {hint}
        </Text>
      </View>
      <Switch accessibilityLabel={label} value={value} onValueChange={onChange} />
    </View>
  );
}

export function PrivacyScreen() {
  const queryClient = useQueryClient();
  const toast = useToast();
  const quickSave = useMutation({
    mutationFn: (input: ProfileInput) => profileApi.save(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
    },
    onError: (e) => toast.show(describeError(e), "error")
  });
  return (
    <WithProfile title="Confidentialité">
      {(p) => {
        const base: ProfileInput = {
          firstName: p.firstName,
          birthDate: p.birthDate,
          gender: p.gender,
          bio: p.bio,
          city: p.city,
          interestIds: p.interests.map((i) => i.id)
        };
        return (
          <View style={styles.stack}>
            <ToggleRow
              label="Profil visible"
              hint="Désactivé, vous n'apparaissez plus dans la découverte. Vos matchs actuels restent actifs."
              value={p.isVisible}
              onChange={(v) => quickSave.mutate({ ...base, isVisible: v, showDistance: p.showDistance })}
            />
            <ToggleRow
              label="Afficher ma distance"
              hint="Les autres voient une distance approximative, jamais votre position exacte."
              value={p.showDistance}
              onChange={(v) => quickSave.mutate({ ...base, isVisible: p.isVisible, showDistance: v })}
            />
            <Text variant="caption" muted>
              Votre position est arrondie à environ 1 km avant d'être enregistrée et n'est jamais communiquée aux autres membres.
            </Text>
          </View>
        );
      }}
    </WithProfile>
  );
}

export function BlockedScreen() {
  const { colors } = useTheme();
  const nav = useNav();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.blocks, queryFn: safetyApi.blocks });
  const unblock = useMutation({
    mutationFn: (userId: string) => safetyApi.unblock(userId),
    onSuccess: () => {
      toast.show("Personne débloquée.");
      void queryClient.invalidateQueries({ queryKey: keys.blocks });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
    },
    onError: (e) => toast.show(describeError(e), "error")
  });
  return (
    <Screen title="Personnes bloquées" onBack={nav.back}>
      {query.isPending ? <LoadingView /> : null}
      {query.isError ? <ErrorState message={describeError(query.error)} onRetry={() => void query.refetch()} /> : null}
      {query.isSuccess && query.data.length === 0 ? (
        <EmptyState title="Personne n'est bloqué" message="Les personnes que vous bloquez apparaîtront ici." />
      ) : null}
      {query.isSuccess && query.data.length > 0 ? (
        <FlatList
          data={query.data}
          keyExtractor={(b) => b.userId}
          renderItem={({ item }) => (
            <View style={[styles.blockRow, { borderBottomColor: colors.border }]}>
              <Text variant="label" style={styles.flex}>
                {item.firstName || "Compte supprimé"}
              </Text>
              <Button title="Débloquer" variant="secondary" onPress={() => unblock.mutate(item.userId)} />
            </View>
          )}
        />
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  stack: { gap: 20 },
  toggle: { flexDirection: "row", alignItems: "center", gap: 12 },
  blockRow: { flexDirection: "row", alignItems: "center", gap: 12, padding: 16, borderBottomWidth: StyleSheet.hairlineWidth }
});
