import React from "react";
import { StyleSheet } from "react-native";
import { PhotoGrid } from "../components/PhotoGrid";
import { Screen, Text, spacing } from "../ui";

export default function PhotosScreen() {
  return (
    <Screen scroll testID="photos-screen">
      <Text style={styles.intro} tone="muted">
        Jusqu'à 6 photos (JPEG ou PNG). Touchez une photo pour la définir comme principale, la déplacer, la remplacer ou
        la supprimer.
      </Text>
      <PhotoGrid />
    </Screen>
  );
}

const styles = StyleSheet.create({ intro: { marginVertical: spacing.lg } });
