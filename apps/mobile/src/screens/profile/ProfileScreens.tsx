import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React from "react";

import { useProfileStatus } from "../../hooks";
import type { MainStackParamList } from "../../navigation/types";
import { Button } from "../../ui/Button";
import { Screen } from "../../ui/Screen";
import { ErrorState, Loading } from "../../ui/States";
import { Text } from "../../ui/Text";
import { PhotosManager } from "./PhotosManager";
import { PreferencesForm } from "./PreferencesScreen";
import { ProfileForm } from "./ProfileFormScreen";

type Props<K extends keyof MainStackParamList> = NativeStackScreenProps<MainStackParamList, K>;

/** Photos step of onboarding. At least one photo is required to enter the app. */
export function PhotosSetupScreen({ onDone }: { onDone: () => void }) {
  const status = useProfileStatus();
  if (status.isLoading) return <Loading />;
  if (status.isError)
    return <ErrorState error={status.error} onRetry={() => void status.refetch()} />;
  const photos = status.data?.profile?.photos ?? [];
  return (
    <Screen scroll>
      <Text variant="title">Vos plus belles photos</Text>
      <Text tone="muted">
        Étape 3 sur 3. Ajoutez au moins une photo (6 maximum). La première sera votre photo
        principale.
      </Text>
      <PhotosManager photos={photos} />
      <Button
        label="Terminer"
        onPress={onDone}
        disabled={photos.length === 0}
        testID="photos-done"
      />
    </Screen>
  );
}

export function EditProfileScreen({ navigation }: Props<"EditProfile">) {
  const status = useProfileStatus();
  if (status.isLoading) return <Loading />;
  if (status.isError)
    return <ErrorState error={status.error} onRetry={() => void status.refetch()} />;
  return (
    <Screen scroll>
      <ProfileForm
        initial={status.data?.profile}
        submitLabel="Enregistrer"
        onSaved={() => navigation.goBack()}
      />
    </Screen>
  );
}

export function PhotosScreen() {
  const status = useProfileStatus();
  if (status.isLoading) return <Loading />;
  if (status.isError)
    return <ErrorState error={status.error} onRetry={() => void status.refetch()} />;
  return (
    <Screen scroll>
      <Text tone="muted">
        Touchez une photo pour la déplacer, la remplacer ou la supprimer. Gardez au moins une photo.
      </Text>
      <PhotosManager photos={status.data?.profile?.photos ?? []} />
    </Screen>
  );
}

export function PreferencesScreen({ navigation }: Props<"Preferences">) {
  const status = useProfileStatus();
  if (status.isLoading) return <Loading />;
  if (status.isError)
    return <ErrorState error={status.error} onRetry={() => void status.refetch()} />;
  return (
    <Screen scroll>
      <PreferencesForm
        initial={status.data?.preferences ?? null}
        submitLabel="Enregistrer"
        onSaved={() => navigation.goBack()}
      />
    </Screen>
  );
}
