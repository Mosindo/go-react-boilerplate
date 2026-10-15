import React from "react";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { PhotoManager } from "../components/PhotoManager";
import { useSetPhotosInCache } from "../hooks/useMyProfileCache";
import { useMyProfile } from "../hooks/queries";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Text } from "../shared/ui";

export default function PhotosScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const query = useMyProfile();
  const setPhotos = useSetPhotosInCache();
  const photos = query.data?.profile?.photos;

  return (
    <ScreenContainer scroll testID="photos-screen">
      <Header onBack={() => navigation.goBack()} title="Mes photos" />
      <Text tone="muted">Touchez une photo pour la définir comme principale, la déplacer, la remplacer ou la supprimer.</Text>
      {query.isError ? <ErrorView error={query.error} onRetry={() => void query.refetch()} /> : null}
      {!photos && !query.isError ? <LoadingView /> : null}
      {photos ? <PhotoManager onChanged={setPhotos} photos={photos} /> : null}
    </ScreenContainer>
  );
}
