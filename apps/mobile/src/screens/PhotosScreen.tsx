import React from "react";
import { errorMessage } from "../api/client";
import { PhotoManager } from "../components/PhotoManager";
import { useMe } from "../hooks/useAuth";
import type { MainScreenProps } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";

export default function PhotosScreen(_props: MainScreenProps<"Photos">) {
  const meQuery = useMe();
  if (meQuery.isPending) {
    return <LoadingView />;
  }
  if (!meQuery.data?.profile) {
    return <ErrorView message={errorMessage(meQuery.error)} onRetry={() => void meQuery.refetch()} />;
  }
  return (
    <ScreenContainer scroll underHeader>
      <PhotoManager photos={meQuery.data.profile.photos} />
    </ScreenContainer>
  );
}
