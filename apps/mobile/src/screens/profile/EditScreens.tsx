import React from "react";
import { useNavigation } from "@react-navigation/native";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { InterestsPicker } from "../../features/InterestsPicker";
import { PhotosManager } from "../../features/PhotosManager";
import { PreferencesForm } from "../../features/PreferencesForm";
import { ProfileForm } from "../../features/ProfileForm";
import { useMyProfile, useProfileMutation } from "../../hooks/queries";
import { LoadingState, Screen, Text, useFeedback } from "../../ui";

function useSaved() {
  const navigation = useNavigation();
  const { toast } = useFeedback();
  return () => {
    toast("Modifications enregistrées", "success");
    navigation.goBack();
  };
}

export function EditProfileScreen() {
  const { data } = useMyProfile();
  const save = useProfileMutation(profileApi.saveProfile);
  const done = useSaved();
  if (!data) {
    return <LoadingState />;
  }
  return (
    <Screen edges={[]}>
      <ProfileForm
        initial={data.profile}
        submitLabel="Enregistrer"
        loading={save.isPending}
        error={save.isError ? errorMessage(save.error) : null}
        onSubmit={(input) => save.mutate(input, { onSuccess: done })}
      />
    </Screen>
  );
}

export function EditPreferencesScreen() {
  const { data } = useMyProfile();
  const save = useProfileMutation(profileApi.savePreferences);
  const done = useSaved();
  if (!data) {
    return <LoadingState />;
  }
  return (
    <Screen edges={[]}>
      <PreferencesForm
        initial={data.preferences}
        submitLabel="Enregistrer"
        loading={save.isPending}
        error={save.isError ? errorMessage(save.error) : null}
        onSubmit={(input) => save.mutate(input, { onSuccess: done })}
      />
    </Screen>
  );
}

export function EditInterestsScreen() {
  const { data } = useMyProfile();
  const save = useProfileMutation(profileApi.saveInterests);
  const done = useSaved();
  if (!data) {
    return <LoadingState />;
  }
  return (
    <Screen edges={[]}>
      <InterestsPicker
        initial={data.interests}
        submitLabel="Enregistrer"
        loading={save.isPending}
        error={save.isError ? errorMessage(save.error) : null}
        onSubmit={(slugs) => save.mutate(slugs, { onSuccess: done })}
      />
    </Screen>
  );
}

export function EditPhotosScreen() {
  const { data } = useMyProfile();
  if (!data) {
    return <LoadingState />;
  }
  return (
    <Screen edges={[]}>
      <Text tone="muted">Touche une photo pour la définir comme principale, la remplacer ou la supprimer.</Text>
      <PhotosManager photos={data.photos} />
    </Screen>
  );
}
