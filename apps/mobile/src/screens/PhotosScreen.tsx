import React from "react";
import { PhotoManager } from "../components/PhotoManager";
import { useOwnProfile } from "../hooks/useOwnProfile";
import { Screen } from "../shared/layout";
import { ErrorView, LoadingView } from "../shared/feedback";

export default function PhotosScreen() {
  const { data, isLoading, isError, refetch } = useOwnProfile();
  if (isLoading)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );
  if (isError || !data)
    return (
      <Screen>
        <ErrorView actionLabel="Réessayer" message="Photos indisponibles." onAction={() => void refetch()} />
      </Screen>
    );
  return (
    <Screen scroll>
      <PhotoManager photos={data.photos} />
    </Screen>
  );
}
