import React from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import { PhotosEditor } from "../features/PhotosEditor";
import { PreferencesForm } from "../features/PreferencesForm";
import { ProfileForm } from "../features/ProfileForm";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { ErrorState, LoadingState, Screen, Text, useToast } from "../ui";

export function EditProfileScreen({
  navigation
}: NativeStackScreenProps<MainStackParams, "EditProfile">) {
  const toast = useToast();
  const q = useQuery({ queryKey: queryKeys.overview, queryFn: profileApi.overview });
  if (q.isLoading) return <LoadingState />;
  if (q.isError || !q.data?.profile)
    return <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />;
  return (
    <Screen scroll>
      <Text variant="heading">Photos</Text>
      <PhotosEditor photos={q.data.profile.photos} />
      <Text variant="heading">Mes informations</Text>
      <ProfileForm
        initial={q.data.profile}
        submitLabel="Enregistrer"
        onSaved={() => {
          toast.show("Profil mis à jour.");
          navigation.goBack();
        }}
      />
    </Screen>
  );
}

export function PreferencesScreen({
  navigation
}: NativeStackScreenProps<MainStackParams, "Preferences">) {
  const toast = useToast();
  const q = useQuery({ queryKey: queryKeys.overview, queryFn: profileApi.overview });
  if (q.isLoading) return <LoadingState />;
  if (q.isError)
    return <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />;
  return (
    <Screen scroll>
      <PreferencesForm
        initial={q.data?.preferences}
        submitLabel="Enregistrer"
        onSaved={() => {
          toast.show("Préférences enregistrées.");
          navigation.goBack();
        }}
      />
    </Screen>
  );
}
