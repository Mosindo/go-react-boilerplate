import React from "react";
import { useAuth } from "../../providers/AuthProvider";
import { errorMessage } from "../../lib/format";
import { Button, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { PhotoGrid } from "./PhotoGrid";
import { useMyProfile } from "./queries";

export function EditPhotosScreen() {
  const profile = useMyProfile();
  const { refreshUser } = useAuth();
  return (
    <Screen scroll header={<Header back title="Mes photos" />}>
      {profile.isLoading ? (
        <LoadingState />
      ) : profile.isError || !profile.data ? (
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      ) : (
        <>
          <Text tone="muted">Touche une photo pour la définir comme principale, la remplacer ou la supprimer.</Text>
          <PhotoGrid photos={profile.data.photos} />
          <Button label="Terminé" onPress={() => void refreshUser()} variant="secondary" />
        </>
      )}
    </Screen>
  );
}
