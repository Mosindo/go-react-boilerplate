import React, { useState } from "react";
import { Alert, Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import * as ImageManipulator from "expo-image-manipulator";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deletePhoto, reorderPhotos, uploadPhoto } from "../api/profile";
import type { OwnProfile, PhotoRef } from "../api/types";
import { errorMessage } from "../lib/errors";
import { PhotoImage, Text, useToast } from "../ui";
import { MIN_TOUCH, radius, spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";

export const MAX_PHOTOS = 6;

type Props = { photos: PhotoRef[] };

/** Picks an image, normalises it to a ≤1440px JPEG on-device (also converts HEIC) and uploads it. */
async function pickAndPrepare(): Promise<string | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) throw new Error("permission");
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    quality: 1,
    allowsEditing: false
  });
  const asset = result.canceled ? undefined : result.assets[0];
  if (!asset) return null;
  const resize =
    asset.width >= asset.height
      ? { width: Math.min(asset.width, 1440) }
      : { height: Math.min(asset.height, 1440) };
  const out = await ImageManipulator.manipulateAsync(asset.uri, [{ resize }], {
    compress: 0.8,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return out.uri;
}

export function PhotosManager({ photos }: Props) {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);

  const apply = (next: PhotoRef[]) =>
    queryClient.setQueryData<OwnProfile | null>(["profile"], (profile) =>
      profile ? { ...profile, photos: next } : profile
    );

  const run = async (task: () => Promise<PhotoRef[] | null>) => {
    setBusy(true);
    try {
      const next = await task();
      if (next) apply(next);
    } catch (error) {
      if (error instanceof Error && error.message === "permission") {
        toast.show("Autorisez l'accès à vos photos dans les réglages.", "error");
      } else {
        toast.show(errorMessage(error), "error");
      }
    } finally {
      setBusy(false);
    }
  };

  const add = (position?: number) =>
    run(async () => {
      const uri = await pickAndPrepare();
      return uri ? uploadPhoto(uri, position) : null;
    });

  const makePrimary = useMutation({
    mutationFn: (id: string) => reorderPhotos([id, ...photos.filter((p) => p.id !== id).map((p) => p.id)]),
    onSuccess: apply,
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const move = (index: number, direction: -1 | 1) => {
    const ids = photos.map((p) => p.id);
    const target = index + direction;
    const a = ids[index];
    const b = ids[target];
    if (a === undefined || b === undefined) return;
    ids[index] = b;
    ids[target] = a;
    void run(() => reorderPhotos(ids));
  };

  const confirmDelete = (photo: PhotoRef) =>
    Alert.alert("Supprimer cette photo ?", "Cette action est définitive.", [
      { text: "Annuler", style: "cancel" },
      { text: "Supprimer", style: "destructive", onPress: () => void run(() => deletePhoto(photo.id)) }
    ]);

  const openMenu = (photo: PhotoRef, index: number) => {
    const actions = [
      ...(index > 0
        ? [{ text: "Définir comme photo principale", onPress: () => makePrimary.mutate(photo.id) }]
        : []),
      ...(index > 0 ? [{ text: "Déplacer vers la gauche", onPress: () => move(index, -1) }] : []),
      ...(index < photos.length - 1
        ? [{ text: "Déplacer vers la droite", onPress: () => move(index, 1) }]
        : []),
      { text: "Remplacer", onPress: () => void add(index) },
      { text: "Supprimer", style: "destructive" as const, onPress: () => confirmDelete(photo) },
      { text: "Annuler", style: "cancel" as const }
    ];
    Alert.alert("Photo", undefined, actions);
  };

  return (
    <View style={styles.grid}>
      {photos.map((photo, index) => (
        <Pressable
          key={photo.id}
          accessibilityRole="button"
          accessibilityLabel={
            index === 0 ? "Photo principale, ouvrir les options" : `Photo ${index + 1}, ouvrir les options`
          }
          onPress={() => openMenu(photo, index)}
          disabled={busy}
          style={styles.cell}
          testID={`photo-${index}`}
        >
          <PhotoImage photoId={photo.id} style={styles.image} />
          {index === 0 ? (
            <View style={[styles.badge, { backgroundColor: colors.accent }]}>
              <Text variant="caption" tone="onAccent" bold>
                Principale
              </Text>
            </View>
          ) : null}
        </Pressable>
      ))}
      {photos.length < MAX_PHOTOS ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Ajouter une photo"
          onPress={() => void add()}
          disabled={busy}
          style={[styles.cell, styles.add, { borderColor: colors.border, backgroundColor: colors.surface }]}
          testID="add-photo"
        >
          <Text variant="title" tone="accent">
            {busy ? "…" : "+"}
          </Text>
          <Text variant="caption" tone="muted">
            Ajouter
          </Text>
        </Pressable>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: {
    width: "31.5%",
    aspectRatio: 3 / 4,
    borderRadius: radius.md,
    overflow: "hidden",
    minHeight: MIN_TOUCH
  },
  image: { width: "100%", height: "100%" },
  add: { borderWidth: 2, borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  badge: {
    position: "absolute",
    left: spacing.xs,
    bottom: spacing.xs,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2
  }
});
