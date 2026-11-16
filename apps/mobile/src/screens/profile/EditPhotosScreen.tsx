import React from "react";
import type { AppScreenProps } from "../../navigation/types";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useMyProfile } from "../../hooks/queries";
import { Header, LoadingState, Screen, Text } from "../../ui";

export default function EditPhotosScreen({ navigation }: AppScreenProps<"EditPhotos">) {
  const me = useMyProfile();
  return (
    <Screen scroll>
      <Header title="Mes photos" onBack={navigation.goBack} />
      {me.data?.profile ? (
        <>
          <Text tone="muted" style={{ marginBottom: 16 }}>
            Touche une photo pour la définir comme principale, la remplacer ou la supprimer. Au moins une photo est nécessaire pour être visible.
          </Text>
          <PhotoGrid photos={me.data.profile.photos} />
        </>
      ) : (
        <LoadingState />
      )}
    </Screen>
  );
}
