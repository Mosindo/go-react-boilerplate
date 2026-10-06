import React from "react";
import { ScrollView, StyleSheet } from "react-native";
import { PhotoManager } from "../components/forms/PhotoManager";
import { useAuth } from "../hooks/useAuth";
import { Notice, spacing } from "../shared/ui";

export default function PhotosScreen() {
  const { self } = useAuth();
  const photos = self?.photos ?? [];
  return (
    <ScrollView contentContainerStyle={styles.content} testID="photos-screen">
      <PhotoManager photos={photos} />
      {photos.length === 0 ? (
        <Notice message="Sans photo, votre profil n'est pas visible des autres." tone="warning" style={styles.notice} />
      ) : null}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.xl, gap: spacing.lg },
  notice: { marginTop: spacing.lg }
});
