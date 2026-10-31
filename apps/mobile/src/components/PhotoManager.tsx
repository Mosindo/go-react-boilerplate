import React from "react";
import { Alert, Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deletePhoto, replacePhoto, reorderPhotos, uploadPhoto } from "../api/dating";
import { ApiError } from "../api/client";
import type { OwnProfile, PhotoRef } from "../api/types";
import { queryKeys } from "../lib/queryKeys";
import { Button, Notice, Text, colors, radii, spacing } from "../shared/ui";
import { RemotePhoto } from "./RemotePhoto";

const MAX_PHOTOS = 6;

type PhotoManagerProps = {
  photos: PhotoRef[];
};

async function pickImage(): Promise<ImagePicker.ImagePickerAsset | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    Alert.alert("Accès aux photos refusé", "Autorisez l'accès à vos photos dans les réglages pour en ajouter.");
    return null;
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 0.85
  });
  return result.canceled ? null : (result.assets[0] ?? null);
}

export function PhotoManager({ photos }: PhotoManagerProps) {
  const queryClient = useQueryClient();

  const apply = (next: PhotoRef[]) => {
    queryClient.setQueryData<OwnProfile>(queryKeys.profile, (old) => (old ? { ...old, photos: next } : old));
    void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
  };

  const add = useMutation({
    mutationFn: async () => {
      const asset = await pickImage();
      return asset ? uploadPhoto(asset.uri, asset.mimeType) : null;
    },
    onSuccess: (next) => next && apply(next)
  });
  const replace = useMutation({
    mutationFn: async (id: string) => {
      const asset = await pickImage();
      return asset ? replacePhoto(id, asset.uri, asset.mimeType) : null;
    },
    onSuccess: (next) => next && apply(next)
  });
  const remove = useMutation({ mutationFn: deletePhoto, onSuccess: apply });
  const reorder = useMutation({ mutationFn: reorderPhotos, onSuccess: apply });

  const busy = add.isPending || replace.isPending || remove.isPending || reorder.isPending;
  const error = [add, replace, remove, reorder].map((m) => m.error).find(Boolean);

  const openActions = (photo: PhotoRef, index: number) => {
    const buttons = [
      ...(index > 0
        ? [
            {
              text: "Définir comme photo principale",
              onPress: () => reorder.mutate([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)])
            }
          ]
        : []),
      { text: "Remplacer", onPress: () => replace.mutate(photo.id) },
      {
        text: "Supprimer",
        style: "destructive" as const,
        onPress: () => remove.mutate(photo.id)
      },
      { text: "Annuler", style: "cancel" as const }
    ];
    Alert.alert("Photo", undefined, buttons);
  };

  return (
    <View style={styles.container}>
      <View style={styles.grid}>
        {photos.map((photo, index) => (
          <Pressable
            accessibilityLabel={index === 0 ? "Photo principale" : `Photo ${index + 1}`}
            accessibilityRole="button"
            disabled={busy}
            key={photo.id}
            onPress={() => openActions(photo, index)}
            style={styles.cell}
          >
            <RemotePhoto style={StyleSheet.absoluteFill} url={photo.url} />
            {index === 0 ? (
              <View style={styles.mainBadge}>
                <Text tone="inverse" variant="caption" weight="bold">
                  Principale
                </Text>
              </View>
            ) : null}
          </Pressable>
        ))}
        {photos.length < MAX_PHOTOS ? (
          <Pressable
            accessibilityLabel="Ajouter une photo"
            accessibilityRole="button"
            disabled={busy}
            onPress={() => add.mutate()}
            style={[styles.cell, styles.addCell]}
          >
            <Text tone="primary" variant="heading" weight="bold">
              +
            </Text>
          </Pressable>
        ) : null}
      </View>
      <Text tone="muted" variant="caption">
        {photos.length}/{MAX_PHOTOS} photos · JPEG ou PNG, 5 Mo maximum. Touchez une photo pour la gérer.
      </Text>
      {busy ? <Text tone="muted">Traitement en cours…</Text> : null}
      {error ? <Notice title={error instanceof ApiError ? error.message : "Action impossible pour le moment."} tone="danger" /> : null}
      {photos.length === 0 ? (
        <Button label="Ajouter ma première photo" loading={add.isPending} onPress={() => add.mutate()} variant="secondary" />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31%", aspectRatio: 3 / 4, borderRadius: radii.md, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  addCell: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    borderStyle: "dashed",
    borderColor: colors.primaryBorder,
    backgroundColor: colors.primarySoft
  },
  mainBadge: {
    position: "absolute",
    left: 4,
    bottom: 4,
    backgroundColor: colors.primary,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2
  }
});
