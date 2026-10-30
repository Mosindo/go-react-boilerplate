import React, { useState } from "react";
import { Image, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as ImagePicker from "expo-image-picker";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { photosApi, type PickedImage } from "../api/endpoints";
import type { Photo } from "../api/types";
import { mediaUrl } from "../config";
import { errorMessage, keys } from "../hooks/queries";
import { LoadingView, Sheet, SheetAction, Text, radii, spacing, useTheme, useToast } from "../ui";

const MAX_PHOTOS = 6;
const PHOTOS_KEY = ["photos"] as const;

async function pickImage(): Promise<PickedImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Autorisez l'accès à vos photos pour en ajouter.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 0.85
  });
  if (result.canceled || !result.assets[0]) {
    return null;
  }
  const asset = result.assets[0];
  return { uri: asset.uri, mimeType: asset.mimeType, fileName: asset.fileName };
}

export function PhotoGrid() {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: PHOTOS_KEY, queryFn: photosApi.list });
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<Photo | null>(null);

  const photos = query.data ?? [];

  const apply = async (action: () => Promise<Photo[]>) => {
    setBusy(true);
    try {
      queryClient.setQueryData(PHOTOS_KEY, await action());
      void queryClient.invalidateQueries({ queryKey: keys.profile });
    } catch (error) {
      toast.show(errorMessage(error, "L'opération a échoué."), "error");
    } finally {
      setBusy(false);
    }
  };

  const add = async () => {
    try {
      const image = await pickImage();
      if (image) await apply(() => photosApi.upload(image));
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const replace = async (photo: Photo) => {
    setSelected(null);
    try {
      const image = await pickImage();
      if (image) await apply(() => photosApi.replace(photo.id, image));
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const move = (photo: Photo, toIndex: number) => {
    setSelected(null);
    const ids = photos.map((p) => p.id).filter((id) => id !== photo.id);
    ids.splice(toIndex, 0, photo.id);
    void apply(() => photosApi.reorder(ids));
  };

  const remove = (photo: Photo) => {
    setSelected(null);
    void apply(() => photosApi.remove(photo.id));
  };

  if (query.isLoading) return <LoadingView label="Chargement des photos…" />;

  const index = selected ? photos.findIndex((p) => p.id === selected.id) : -1;

  return (
    <View style={styles.grid} testID="photo-grid">
      {Array.from({ length: MAX_PHOTOS }, (_, slot) => {
        const photo = photos[slot];
        if (photo) {
          return (
            <Pressable
              accessibilityLabel={slot === 0 ? "Photo principale, modifier" : `Photo ${slot + 1}, modifier`}
              accessibilityRole="button"
              key={photo.id}
              onPress={() => setSelected(photo)}
              style={[styles.slot, { backgroundColor: colors.surfaceAlt }]}
            >
              <Image accessibilityIgnoresInvertColors source={{ uri: mediaUrl(photo.url) }} style={styles.image} />
              {slot === 0 ? (
                <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                  <Text variant="caption" style={{ color: colors.primaryText }}>
                    Principale
                  </Text>
                </View>
              ) : null}
            </Pressable>
          );
        }
        return (
          <Pressable
            accessibilityLabel="Ajouter une photo"
            accessibilityRole="button"
            disabled={busy || slot !== photos.length}
            key={`empty-${slot}`}
            onPress={add}
            style={[
              styles.slot,
              styles.empty,
              { borderColor: colors.border, opacity: slot === photos.length ? 1 : 0.45 }
            ]}
            testID={slot === photos.length ? "photo-add" : undefined}
          >
            <Ionicons color={colors.primary} name="add" size={30} />
          </Pressable>
        );
      })}
      {busy ? (
        <View style={[styles.busy, { backgroundColor: colors.overlay }]}>
          <LoadingView label="Envoi…" />
        </View>
      ) : null}

      <Sheet onClose={() => setSelected(null)} title="Cette photo" visible={Boolean(selected)}>
        {selected && index > 0 ? (
          <SheetAction label="Définir comme photo principale" onPress={() => move(selected, 0)} />
        ) : null}
        {selected && index > 1 ? (
          <SheetAction label="Avancer d'une place" onPress={() => move(selected, index - 1)} />
        ) : null}
        {selected ? <SheetAction label="Remplacer" onPress={() => replace(selected)} /> : null}
        {selected ? <SheetAction danger label="Supprimer" onPress={() => remove(selected)} /> : null}
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.md },
  slot: { width: "30.5%", aspectRatio: 3 / 4, borderRadius: radii.md, overflow: "hidden" },
  image: { width: "100%", height: "100%" },
  empty: { borderWidth: 1.5, borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  badge: {
    position: "absolute",
    left: spacing.xs,
    bottom: spacing.xs,
    paddingHorizontal: spacing.sm,
    borderRadius: radii.pill
  },
  busy: { ...StyleSheet.absoluteFillObject, borderRadius: radii.md, justifyContent: "center" }
});
