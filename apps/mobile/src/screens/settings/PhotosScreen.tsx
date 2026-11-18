import React from "react";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useMyProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { Card, Header, LoadingState, Screen, Text } from "../../ui";

export default function PhotosScreen({ navigation }: AppScreenProps<"Photos">) {
  const query = useMyProfile();
  return (
    <Screen scroll testID="photos-screen">
      <Header
        onBack={() => navigation.goBack()}
        title="Mes photos"
        subtitle="Touchez une photo pour la déplacer, la remplacer ou la supprimer."
      />
      {query.data ? <PhotoGrid photos={query.data.photos} /> : <LoadingState />}
      <Card>
        <Text variant="bodyStrong">Conseils</Text>
        <Text color="textMuted" variant="caption">
          Une photo principale lumineuse où l’on voit votre sourire, puis des photos qui racontent ce que vous aimez faire. Évitez les
          lunettes de soleil et les photos de groupe.
        </Text>
      </Card>
    </Screen>
  );
}
