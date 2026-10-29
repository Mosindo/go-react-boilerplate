import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import * as ImagePicker from "expo-image-picker";
import React, { useState } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { deletePhoto, reorderPhotos, uploadPhoto } from "../api/profile";
import type { Photo } from "../api/types";
import { queryKeys, usePhotos } from "../hooks/queries";
import { errorMessage } from "../hooks/useAsyncAction";
import { useAuth } from "../state/AuthProvider";
import { useToast } from "../state/ToastProvider";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { AuthImage } from "./AuthImage";
import { Button } from "./Button";
import { Sheet } from "./Sheet";
import { ErrorView, LoadingView } from "./StateViews";

const MAX_PHOTOS = 6;

async function pickImage(): Promise<{ uri: string; name: string; type: string } | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Autorisez l'accès à vos photos dans les réglages pour en ajouter.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.8
  });
  const asset = result.assets?.[0];
  if (result.canceled || !asset) {
    return null;
  }
  return {
    uri: asset.uri,
    name: asset.fileName ?? "photo.jpg",
    type: asset.mimeType ?? "image/jpeg"
  };
}

/** Grid of up to six photos; the first one is the main photo shown on the card. */
export function PhotoGrid() {
  const theme = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { refreshMe } = useAuth();
  const photosQuery = usePhotos();
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<Photo | null>(null);

  const photos = photosQuery.data ?? [];

  const afterChange = async () => {
    await queryClient.invalidateQueries({ queryKey: queryKeys.photos });
    void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
    await refreshMe();
  };

  const run = async (work: () => Promise<void>) => {
    if (busy) return;
    setBusy(true);
    try {
      await work();
      await afterChange();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(false);
    }
  };

  const add = () =>
    run(async () => {
      const file = await pickImage();
      if (file) await uploadPhoto(file);
    });

  const replace = (photo: Photo) => {
    setSelected(null);
    return run(async () => {
      const file = await pickImage();
      if (!file) return;
      const added = await uploadPhoto(file);
      await deletePhoto(photo.id);
      // Put the new photo where the old one was.
      const order = photos.filter((p) => p.id !== photo.id).map((p) => p.id);
      order.splice(photo.position, 0, added.id);
      await reorderPhotos(order);
    });
  };

  const makeMain = (photo: Photo) => {
    setSelected(null);
    return run(async () => {
      await reorderPhotos([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]);
    });
  };

  const remove = (photo: Photo) => {
    setSelected(null);
    return run(async () => {
      await deletePhoto(photo.id);
    });
  };

  if (photosQuery.isLoading) return <LoadingView />;
  if (photosQuery.isError) return <ErrorView onRetry={() => void photosQuery.refetch()} />;

  const slots = Array.from({ length: MAX_PHOTOS }, (_, index) => photos[index] ?? null);

  return (
    <View>
      <View style={styles.grid}>
        {slots.map((photo, index) => (
          <View key={photo?.id ?? `empty-${index}`} style={styles.cell}>
            {photo ? (
              <Pressable
                accessibilityLabel={`Photo ${index + 1}${index === 0 ? ", photo principale" : ""}`}
                accessibilityRole="button"
                onPress={() => setSelected(photo)}
                style={styles.fill}
              >
                <AuthImage style={styles.image} url={photo.url} />
                {index === 0 ? (
                  <View style={[styles.badge, { backgroundColor: theme.primary }]}>
                    <Text style={styles.badgeText}>Principale</Text>
                  </View>
                ) : null}
              </Pressable>
            ) : (
              <Pressable
                accessibilityLabel="Ajouter une photo"
                accessibilityRole="button"
                disabled={busy || index !== photos.length}
                onPress={() => void add()}
                style={[
                  styles.fill,
                  styles.empty,
                  {
                    borderColor: theme.border,
                    backgroundColor: theme.surface,
                    opacity: index === photos.length ? 1 : 0.4
                  }
                ]}
              >
                <Ionicons color={theme.primary} name="add" size={32} />
              </Pressable>
            )}
          </View>
        ))}
      </View>
      <Text style={[styles.hint, { color: theme.textMuted }]}>
        {photos.length}/{MAX_PHOTOS} photos · La première est votre photo principale. Les photos
        sont compressées et nettoyées de leurs métadonnées.
      </Text>
      {busy ? <LoadingView label="Traitement de la photo…" /> : null}

      <Sheet onClose={() => setSelected(null)} title="Cette photo" visible={selected !== null}>
        {selected ? (
          <View style={styles.actions}>
            {selected.position !== 0 ? (
              <Button
                onPress={() => void makeMain(selected)}
                title="Définir comme principale"
                variant="secondary"
              />
            ) : null}
            <Button onPress={() => void replace(selected)} title="Remplacer" variant="secondary" />
            <Button
              disabled={photos.length <= 1}
              onPress={() => void remove(selected)}
              title={photos.length <= 1 ? "Gardez au moins une photo" : "Supprimer"}
              variant="danger"
            />
          </View>
        ) : null}
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", marginHorizontal: -spacing.xs },
  cell: { width: "33.33%", aspectRatio: 0.8, padding: spacing.xs },
  fill: { flex: 1, borderRadius: radii.md, overflow: "hidden" },
  image: { flex: 1 },
  empty: {
    borderWidth: 1.5,
    borderStyle: "dashed",
    alignItems: "center",
    justifyContent: "center"
  },
  badge: {
    position: "absolute",
    left: 6,
    bottom: 6,
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 3
  },
  badgeText: { color: "#FFF", fontSize: 11, fontWeight: "700" },
  hint: { fontSize: fontSizes.xs, lineHeight: 18, marginTop: spacing.md },
  actions: { gap: spacing.sm }
});
