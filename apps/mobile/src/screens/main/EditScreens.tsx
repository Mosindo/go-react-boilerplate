import React from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { savePreferences, saveProfile } from "../../api/profile";
import type { OwnProfile } from "../../api/types";
import { PhotosManager } from "../../components/PhotosManager";
import { DEFAULT_PREFERENCES, PreferencesForm } from "../../components/PreferencesForm";
import { ProfileForm } from "../../components/ProfileForm";
import { useOwnProfile } from "../../hooks/queries";
import { emitDiscoverReset } from "../../lib/discoverEvents";
import type { MainStackParams } from "../../navigation/types";
import { ErrorState, Loading, Screen, Text, useToast } from "../../ui";

export function EditProfileScreen({ navigation }: NativeStackScreenProps<MainStackParams, "EditProfile">) {
  const query = useOwnProfile();
  const queryClient = useQueryClient();
  const toast = useToast();
  if (query.isPending) return <Loading />;
  if (query.isError || !query.data)
    return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  const profile = query.data;
  return (
    <Screen scroll testID="edit-profile">
      <ProfileForm
        initial={profile}
        submitLabel="Enregistrer"
        onSubmit={async (input) => {
          const saved = await saveProfile({ ...input, isVisible: profile.isVisible });
          queryClient.setQueryData(["profile"], saved);
          toast.show("Profil mis à jour.", "success");
          navigation.goBack();
        }}
      />
    </Screen>
  );
}

export function PhotosScreen() {
  const query = useOwnProfile();
  if (query.isPending) return <Loading />;
  if (query.isError || !query.data)
    return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <Screen scroll testID="photos-screen">
      <Text tone="muted">
        Touchez une photo pour la déplacer, la remplacer, la définir comme principale ou la supprimer. Les
        photos sont réencodées et nettoyées de leurs métadonnées (GPS inclus).
      </Text>
      <PhotosManager photos={query.data.photos} />
    </Screen>
  );
}

export function PreferencesScreen({ navigation }: NativeStackScreenProps<MainStackParams, "Preferences">) {
  const query = useOwnProfile();
  const queryClient = useQueryClient();
  const toast = useToast();
  if (query.isPending) return <Loading />;
  if (query.isError || !query.data)
    return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <Screen scroll testID="preferences-screen">
      <PreferencesForm
        initial={query.data.preferences ?? DEFAULT_PREFERENCES}
        submitLabel="Enregistrer"
        onSubmit={async (prefs) => {
          const saved = await savePreferences(prefs);
          queryClient.setQueryData<OwnProfile | null>(["profile"], (current) =>
            current ? { ...current, preferences: saved } : current
          );
          // The candidate pool changed: let Discover refill from scratch.
          emitDiscoverReset();
          toast.show("Préférences enregistrées.", "success");
          navigation.goBack();
        }}
      />
    </Screen>
  );
}
