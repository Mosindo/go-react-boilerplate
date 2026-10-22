import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { deletePhoto, replacePhoto, reorderPhotos, setPrimaryPhoto, uploadPhoto } from "../api/profile";
import type { PhotoRef } from "../api/types";
import { usePhotosQuery } from "../hooks/useProfileData";
import { pickPhoto } from "../lib/images";
import { MAX_PHOTOS } from "../lib/validation";
import { Loader, Notice, Text, colors, radii, spacing } from "../shared/ui";
import { ActionSheet } from "./Dialogs";
import { AuthImage } from "./AuthImage";

/** Grid of up to six photo slots: add, replace, delete, reorder, choose the main photo. */
export function PhotoManager() {
  const queryClient = useQueryClient();
  const photos = usePhotosQuery();
  const [selected, setSelected] = useState<PhotoRef | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["photos"] }),
      queryClient.invalidateQueries({ queryKey: ["profile"] }),
      queryClient.invalidateQueries({ queryKey: ["discover"] })
    ]);
  };

  const run = useMutation({
    mutationFn: async (task: () => Promise<unknown>) => {
      setError(null);
      await task();
    },
    onSuccess: refresh,
    onError: (err) => setError(errorMessage(err, "Action impossible."))
  });

  const add = () =>
    run.mutate(async () => {
      const image = await pickPhoto();
      if (image) {
        await uploadPhoto(image);
      }
    });

  const list = photos.data ?? [];
  const actions = selected
    ? [
        ...(selected.position > 0
          ? [
              {
                label: "Définir comme photo principale",
                onPress: () => run.mutate(() => setPrimaryPhoto(selected.id)),
                testID: "photo-primary"
              },
              {
                label: "Avancer d'une place",
                onPress: () => {
                  const ids = list.map((p) => p.id);
                  const i = ids.indexOf(selected.id);
                  [ids[i - 1], ids[i]] = [ids[i], ids[i - 1]];
                  run.mutate(() => reorderPhotos(ids));
                }
              }
            ]
          : []),
        {
          label: "Remplacer",
          onPress: () =>
            run.mutate(async () => {
              const image = await pickPhoto();
              if (image) {
                await replacePhoto(selected.id, image);
              }
            })
        },
        { label: "Supprimer", destructive: true, onPress: () => run.mutate(() => deletePhoto(selected.id)), testID: "photo-delete" }
      ]
    : [];

  return (
    <View style={styles.root}>
      <View style={styles.grid}>
        {Array.from({ length: MAX_PHOTOS }, (_, slot) => {
          const photo = list[slot];
          return photo ? (
            <Pressable
              accessibilityLabel={slot === 0 ? "Photo principale" : `Photo ${slot + 1}`}
              accessibilityRole="button"
              key={photo.id}
              onPress={() => setSelected(photo)}
              style={styles.slot}
              testID={`photo-slot-${slot}`}
            >
              <AuthImage photo={photo} size="thumb" style={StyleSheet.absoluteFill} />
              {slot === 0 ? (
                <View style={styles.mainTag}>
                  <Text tone="inverse" variant="caption" weight="bold">
                    Principale
                  </Text>
                </View>
              ) : null}
            </Pressable>
          ) : (
            <Pressable
              accessibilityLabel="Ajouter une photo"
              accessibilityRole="button"
              disabled={run.isPending || slot > list.length}
              key={`empty-${slot}`}
              onPress={add}
              style={[styles.slot, styles.empty, slot > list.length ? styles.dimmed : null]}
              testID={slot === list.length ? "add-photo" : undefined}
            >
              <Text tone="primary" variant="heading" weight="bold">
                +
              </Text>
            </Pressable>
          );
        })}
      </View>
      {photos.isLoading || run.isPending ? <Loader label={run.isPending ? "Envoi en cours…" : undefined} size="small" /> : null}
      {error ? <Notice description={error} title="Photo non enregistrée" tone="danger" /> : null}
      <ActionSheet onClose={() => setSelected(null)} options={actions} title="Cette photo" visible={selected !== null} />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: { width: "31.5%", aspectRatio: 0.8, borderRadius: radii.md, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  empty: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1.5,
    borderStyle: "dashed",
    borderColor: colors.primaryBorder,
    backgroundColor: colors.surfaceAccent
  },
  dimmed: { opacity: 0.45 },
  mainTag: {
    position: "absolute",
    left: 6,
    bottom: 6,
    backgroundColor: "rgba(20,10,16,0.65)",
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 2
  }
});
