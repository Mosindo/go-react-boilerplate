import React, { useState } from "react";
import { Image, Pressable, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AppText, Banner, Button, Row } from "./ui";
import { profileApi } from "../api/services";
import { errorMessage } from "../api/http";
import { mediaUrl } from "../api/config";
import { radius, spacing, useTheme } from "../theme/theme";
import type { PhotoView } from "../api/types";

const MAX = 6;

export default function PhotosEditor({ photos }: { photos: PhotoView[] }) {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const refresh = () => qc.invalidateQueries({ queryKey: ["profile"] });

  const upload = useMutation({
    mutationFn: (asset: ImagePicker.ImagePickerAsset) => profileApi.uploadPhoto(asset.uri, asset.mimeType),
    onSuccess: refresh,
    onError: (e) => setError(errorMessage(e)),
  });
  const remove = useMutation({
    mutationFn: (id: string) => profileApi.deletePhoto(id),
    onSuccess: refresh,
    onError: (e) => setError(errorMessage(e)),
  });
  const reorder = useMutation({
    mutationFn: (ids: string[]) => profileApi.reorderPhotos(ids),
    onSuccess: refresh,
    onError: (e) => setError(errorMessage(e)),
  });

  const pick = async () => {
    setError(null);
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted && perm.canAskAgain === false) {
      setError("Accès aux photos refusé. Autorisez-le dans les réglages de votre téléphone.");
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"],
      quality: 0.85,
      allowsEditing: true,
      aspect: [3, 4],
    });
    const asset = result.assets?.[0];
    if (!result.canceled && asset) upload.mutate(asset);
  };

  const makeMain = (id: string) =>
    reorder.mutate([id, ...photos.filter((p) => p.id !== id).map((p) => p.id)]);
  const busy = upload.isPending || remove.isPending || reorder.isPending;

  return (
    <View style={{ gap: spacing.md }}>
      {error ? <Banner message={error} /> : null}
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.md }}>
        {photos.map((p, index) => (
          <View key={p.id} testID={`photo-${index}`} style={{ width: "30%", flexGrow: 1, gap: spacing.xs }}>
            <Image
              accessibilityLabel={`Photo ${index + 1}${index === 0 ? " (principale)" : ""}`}
              source={{ uri: mediaUrl(p.url) }}
              style={{ aspectRatio: 3 / 4, borderRadius: radius.md, backgroundColor: colors.surfaceAlt }}
            />
            <Row style={{ justifyContent: "space-between", gap: spacing.xs }}>
              {index === 0 ? (
                <AppText variant="muted">Principale</AppText>
              ) : (
                <Pressable
                  accessibilityRole="button"
                  accessibilityLabel="Définir comme photo principale"
                  onPress={() => makeMain(p.id)}
                  disabled={busy}
                >
                  <AppText variant="small" style={{ color: colors.primary, fontWeight: "700" }}>
                    ★ Principale
                  </AppText>
                </Pressable>
              )}
              <Pressable
                testID={`delete-photo-${index}`}
                accessibilityRole="button"
                accessibilityLabel="Supprimer la photo"
                onPress={() => remove.mutate(p.id)}
                disabled={busy}
              >
                <AppText variant="error">Supprimer</AppText>
              </Pressable>
            </Row>
          </View>
        ))}
      </View>
      {photos.length < MAX ? (
        <Button
          testID="add-photo"
          title={photos.length === 0 ? "Ajouter ma première photo" : "Ajouter une photo"}
          variant="secondary"
          loading={upload.isPending}
          onPress={pick}
        />
      ) : null}
      <AppText variant="muted">
        {photos.length}/{MAX} photos · JPEG ou PNG, 5 Mo maximum. Les métadonnées (GPS, etc.) sont supprimées.
      </AppText>
    </View>
  );
}
