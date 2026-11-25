import React from "react";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useMyProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { Header, LoadingState, Screen } from "../../ui";

export function EditPhotosScreen({ navigation }: AppScreenProps<"EditPhotos">) {
  const me = useMyProfile();
  return (
    <Screen scroll testID="edit-photos-screen">
      <Header title="Mes photos" onBack={navigation.goBack} />
      {me.data ? <PhotoGrid photos={me.data.photos} /> : <LoadingState />}
    </Screen>
  );
}
