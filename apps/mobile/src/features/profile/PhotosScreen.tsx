import React from "react";
import { StyleSheet, View } from "react-native";
import { useMyProfile } from "../../hooks";
import { spacing } from "../../theme";
import { LoadingState, Screen, Text } from "../../ui";
import { PhotoGrid } from "./fields";

export function PhotosScreen() {
  const profile = useMyProfile();
  if (!profile.data)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  const count = profile.data.photos.length;
  return (
    <Screen scroll edges={["left", "right", "bottom"]} testID="photos-screen">
      <View style={styles.body}>
        <Text tone="muted">Ajoute jusqu’à 6 photos (JPEG ou PNG, 10 Mo max). La première est ta photo principale.</Text>
        <PhotoGrid photos={profile.data.photos} />
        {count === 0 ? <Text tone="danger">Sans photo, ton profil n’apparaît plus dans Découvrir.</Text> : null}
        <Text variant="caption" tone="subtle">
          Les métadonnées (dont la localisation GPS) sont supprimées de tes photos avant leur enregistrement.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: {
    gap: spacing.lg,
    paddingTop: spacing.md
  }
});
