import React from "react";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useMyProfile } from "../../hooks/queries";
import { LoadingView } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";

export default function PhotosScreen() {
  const profile = useMyProfile();
  if (!profile.data) {
    return <LoadingView />;
  }
  return (
    <Screen testID="photos-screen">
      <ScreenHeader subtitle="Touchez une photo pour la remplacer, la déplacer ou la supprimer." title="Mes photos" />
      <PhotoGrid maxPhotos={profile.data.maxPhotos} photos={profile.data.photos} />
    </Screen>
  );
}
