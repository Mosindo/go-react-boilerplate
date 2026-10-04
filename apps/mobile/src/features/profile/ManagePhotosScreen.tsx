import React from "react";
import { useOwnProfile } from "../../hooks/useProfile";
import { ErrorState, LoadingState, Screen, Text } from "../../ui";
import { PhotoManager } from "./PhotoManager";

export function ManagePhotosScreen() {
  const query = useOwnProfile();
  if (query.isPending) return <LoadingState />;
  if (query.isError || !query.data) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <Screen scroll>
      <Text tone="muted">
        Profiles with several clear photos get more conversations. At least one photo is required.
      </Text>
      <PhotoManager name={query.data.displayName} photos={query.data.photos} />
    </Screen>
  );
}
