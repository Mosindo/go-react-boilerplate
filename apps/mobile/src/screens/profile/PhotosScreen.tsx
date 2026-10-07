import { useQueryClient } from "@tanstack/react-query";
import React from "react";
import { PhotoManager } from "../../features/profile/PhotoManager";
import { profileKeys, useMyProfile } from "../../hooks/useProfile";
import { strings } from "../../i18n/strings";
import { ErrorView, LoadingView } from "../../shared/feedback";
import { FormScreen } from "../../shared/layout";
import { Text } from "../../shared/ui";
import type { Photo, SelfProfile } from "../../api/types";

export default function PhotosScreen() {
  const query = useMyProfile();
  const client = useQueryClient();

  function onChange(photos: Photo[]) {
    client.setQueryData<SelfProfile | null>(profileKeys.me, (current) =>
      current ? { ...current, photos, complete: photos.length > 0 } : current
    );
  }

  return (
    <FormScreen edges={["left", "right", "bottom"]}>
      <Text tone="muted">{strings.photos.intro}</Text>
      {query.isLoading ? <LoadingView /> : null}
      {query.isError ? (
        <ErrorView message={strings.errors.generic} onAction={() => void query.refetch()} />
      ) : null}
      {query.data ? <PhotoManager onChange={onChange} photos={query.data.photos} /> : null}
    </FormScreen>
  );
}
