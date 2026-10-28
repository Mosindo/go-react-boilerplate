import React from "react";
import { PhotosManager, Screen, ErrorState, LoadingState } from "../../components";
import { useOwnProfile } from "../../hooks/queries";
import { errorMessage } from "../../lib/format";

export function PhotosScreen() {
  const query = useOwnProfile();
  if (query.isPending) return <Screen><LoadingState /></Screen>;
  if (query.isError) return <Screen><ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} /></Screen>;
  return (
    <Screen scroll testID="photos-screen">
      <PhotosManager photos={query.data?.photos ?? []} />
    </Screen>
  );
}
