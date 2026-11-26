import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { MyProfile } from "../../api/types";
import { queryClient, queryKeys } from "../../hooks/queryClient";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Header, Screen, Text } from "../../ui";
import { PhotoGrid } from "./components/PhotoGrid";

type Props = NativeStackScreenProps<AppStackParamList, "Photos">;

export function PhotosScreen({ navigation }: Props) {
  const { profile } = useAuth();
  if (!profile) return null;
  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="Mes photos" />
      <View style={styles.body}>
        <Text tone="muted">Touchez une photo pour la définir comme principale, la remplacer ou la supprimer. Jusqu’à 6 photos.</Text>
        <PhotoGrid
          onChange={(photos) => {
            queryClient.setQueryData<MyProfile>(queryKeys.profile, (p) =>
              p
                ? {
                    ...p,
                    photos,
                    completion: {
                      ...p.completion,
                      photos: photos.length > 0,
                      complete: p.completion.profile && p.completion.preferences && photos.length > 0
                    }
                  }
                : p
            );
          }}
          photos={profile.photos}
        />
        <Text tone="subtle" variant="caption">
          Les photos sont recompressées et leurs métadonnées (dont la position GPS) sont supprimées à l’envoi.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ body: { gap: spacing.lg } });
