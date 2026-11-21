import React from "react";
import { Header, LoadingState, Screen, Text } from "../../ui";
import { useMyProfile } from "./hooks";
import { PhotoGrid } from "./PhotoGrid";

export function EditPhotosScreen() {
  const me = useMyProfile().data;
  return (
    <Screen scroll>
      <Header title="Mes photos" />
      {!me ? (
        <LoadingState />
      ) : (
        <>
          <Text tone="muted">Touchez une photo pour la définir comme principale, la remplacer ou la supprimer. 6 photos maximum.</Text>
          <PhotoGrid photos={me.photos} minimum={1} />
          <Text variant="caption" tone="subtle">
            Formats JPEG, PNG ou WebP, 10 Mo maximum. Les images sont recadrées, compressées et nettoyées de leurs métadonnées.
          </Text>
        </>
      )}
    </Screen>
  );
}
