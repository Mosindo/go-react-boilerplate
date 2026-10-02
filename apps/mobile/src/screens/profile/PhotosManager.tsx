import { Ionicons } from "@expo/vector-icons";
import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";

import { ApiError } from "../../api/client";
import { photosApi, type UploadFile } from "../../api/endpoints";
import type { Photo } from "../../api/types";
import { keys } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { useFeedback } from "../../ui/Feedback";
import { PhotoImage } from "../../ui/PhotoImage";
import { Text } from "../../ui/Text";

export const MAX_PHOTOS = 6;

/** Opens the library, then resizes and re-encodes to JPEG on device to save bandwidth. */
async function pickImage(): Promise<UploadFile | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 1,
  });
  const asset = result.canceled ? null : result.assets[0];
  if (!asset) return null;
  const resized = await ImageManipulator.manipulateAsync(
    asset.uri,
    asset.width > 1280 ? [{ resize: { width: 1280 } }] : [],
    { compress: 0.82, format: ImageManipulator.SaveFormat.JPEG },
  );
  return { uri: resized.uri, name: "photo.jpg", type: "image/jpeg" };
}

function uploadError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.code === "invalid_image") return "Format non pris en charge (JPEG, PNG ou WebP).";
    if (err.code === "image_too_small") return "Image trop petite (200 px minimum).";
    if (err.code === "image_too_large") return "Image trop lourde (10 Mo maximum).";
    if (err.code === "photo_limit") return "Vous avez atteint 6 photos.";
    return err.message;
  }
  return "Envoi impossible.";
}

export function PhotosManager({ photos }: { photos: Photo[] }) {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const { toast, sheet, confirm } = useFeedback();
  const [busyId, setBusyId] = useState<string | null>(null);

  const refresh = () => qc.invalidateQueries({ queryKey: keys.profile });
  const onError = (err: unknown) => toast(uploadError(err), "error");

  const add = useMutation({ mutationFn: photosApi.upload, onSuccess: refresh, onError });
  const replace = useMutation({
    mutationFn: ({ id, file }: { id: string; file: UploadFile }) => photosApi.replace(id, file),
    onSuccess: refresh,
    onError,
  });
  const remove = useMutation({ mutationFn: photosApi.remove, onSuccess: refresh, onError });
  const reorder = useMutation({ mutationFn: photosApi.reorder, onSuccess: refresh, onError });
  const primary = useMutation({ mutationFn: photosApi.makePrimary, onSuccess: refresh, onError });

  const addPhoto = async () => {
    const file = await pickImage();
    if (file) add.mutate(file);
  };

  const move = (index: number, delta: number) => {
    const ids = photos.map((p) => p.id);
    const target = index + delta;
    const moved = ids[index];
    const other = ids[target];
    if (moved === undefined || other === undefined) return;
    ids[index] = other;
    ids[target] = moved;
    reorder.mutate(ids);
  };

  const openMenu = (photo: Photo, index: number) => {
    const options = [];
    if (index > 0)
      options.push({
        label: "Définir comme photo principale",
        onPress: () => primary.mutate(photo.id),
      });
    if (index > 0)
      options.push({ label: "Déplacer vers la gauche", onPress: () => move(index, -1) });
    if (index < photos.length - 1)
      options.push({ label: "Déplacer vers la droite", onPress: () => move(index, 1) });
    options.push({
      label: "Remplacer",
      onPress: async () => {
        const file = await pickImage();
        if (file) {
          setBusyId(photo.id);
          replace.mutate({ id: photo.id, file }, { onSettled: () => setBusyId(null) });
        }
      },
    });
    options.push({
      label: "Supprimer",
      destructive: true,
      onPress: async () => {
        if (
          await confirm({
            title: "Supprimer cette photo ?",
            message: "Cette action est définitive.",
            confirmLabel: "Supprimer",
            destructive: true,
          })
        ) {
          remove.mutate(photo.id);
        }
      },
    });
    sheet(index === 0 ? "Photo principale" : `Photo ${index + 1}`, options);
  };

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i]);
  const uploading = add.isPending;

  return (
    <View style={styles.grid}>
      {slots.map((photo, i) =>
        photo ? (
          <Pressable
            key={photo.id}
            accessibilityRole="button"
            accessibilityLabel={`Photo ${i + 1}, options`}
            onPress={() => openMenu(photo, i)}
            style={styles.cell}
          >
            <PhotoImage photo={photo} style={styles.fill} />
            {i === 0 ? (
              <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                <Text variant="caption" tone="onPrimary">
                  Principale
                </Text>
              </View>
            ) : null}
            {busyId === photo.id ? (
              <View style={[styles.fill, styles.busy, { backgroundColor: colors.overlay }]} />
            ) : null}
          </Pressable>
        ) : (
          <Pressable
            key={`empty-${i}`}
            testID={i === photos.length ? "photo-add" : undefined}
            accessibilityRole="button"
            accessibilityLabel="Ajouter une photo"
            disabled={i !== photos.length || uploading}
            onPress={addPhoto}
            style={[
              styles.cell,
              styles.empty,
              { borderColor: colors.border, backgroundColor: colors.surface },
              i !== photos.length && { opacity: 0.45 },
            ]}
          >
            <Ionicons
              name={uploading && i === photos.length ? "hourglass-outline" : "add"}
              size={30}
              color={colors.primary}
            />
          </Pressable>
        ),
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 0.8, borderRadius: radii.md, overflow: "hidden" },
  fill: { width: "100%", height: "100%" },
  empty: { borderWidth: 2, borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  badge: {
    position: "absolute",
    left: 6,
    bottom: 6,
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: radii.pill,
  },
  busy: { position: "absolute" },
});
