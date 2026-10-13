import React from "react";
import { ScrollView, StyleSheet } from "react-native";
import { PhotoGrid } from "../features/profile/PhotoGrid";
import { useOwnProfile } from "../hooks/queries";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Notice, Text, spacing } from "../shared/ui";

export default function PhotosScreen() {
  const query = useOwnProfile();
  if (query.isLoading) {
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  }
  if (!query.data) {
    return (
      <ScreenContainer>
        <ErrorView message="Impossible de charger vos photos." onAction={() => void query.refetch()} />
      </ScreenContainer>
    );
  }
  return (
    <ScreenContainer>
      <ScrollView contentContainerStyle={styles.scroll}>
        <Text tone="muted">Touchez une photo pour la déplacer, la remplacer ou la supprimer.</Text>
        <PhotoGrid photos={query.data.photos} />
        {query.data.photos.length === 0 ? (
          <Notice message="Sans photo, votre profil n'apparaît pas dans la découverte." tone="warning" />
        ) : null}
      </ScrollView>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({ scroll: { gap: spacing.lg, paddingBottom: spacing.xxxl } });
