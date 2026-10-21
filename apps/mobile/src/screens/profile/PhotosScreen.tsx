import React from "react";
import { PhotoGrid } from "../../components/PhotoGrid";
import { usePhotoManager } from "../../hooks/usePhotoManager";
import { ErrorView, LoadingView } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Text } from "../../shared/ui";

export function PhotosScreen() {
  const manager = usePhotoManager();
  if (manager.query.isPending)
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  if (manager.query.isError)
    return (
      <ScreenContainer>
        <ErrorView message={manager.query.error.message} onRetry={() => void manager.query.refetch()} />
      </ScreenContainer>
    );
  return (
    <ScreenContainer scroll edges={["bottom", "left", "right"]} testID="photos-screen">
      <Text tone="muted">
        Touchez une photo pour la déplacer, la remplacer ou la supprimer. La première est votre photo principale. Les photos sont
        compressées et leurs métadonnées (lieu, appareil) supprimées.
      </Text>
      <PhotoGrid
        photos={manager.photos}
        busy={manager.busy}
        onAdd={() => void manager.add()}
        onReplace={(p) => void manager.replace(p)}
        onDelete={(p) => void manager.remove(p)}
        onMakeMain={(p) => manager.move(p, 0)}
        onMove={(p, dir) => manager.move(p, p.position + dir)}
      />
      {manager.photos.length === 1 ? (
        <Text variant="caption" tone="muted">
          Une photo au minimum est nécessaire pour apparaître dans la découverte.
        </Text>
      ) : null}
    </ScreenContainer>
  );
}
