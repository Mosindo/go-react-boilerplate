import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { photosApi } from "../../api/endpoints";
import type { MyProfile, Photo } from "../../api/types";
import { absoluteUrl } from "../../config";
import { queryKeys } from "../../queryClient";
import { colors, radii, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { pickAndCompressPhoto } from "../../utils/media";
import { AppText } from "../AppText";
import { Button } from "../Button";
import { Sheet } from "../Sheet";
import { showToast } from "../Toast";

const SLOTS = 6;

export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const client = useQueryClient();
  const [selected, setSelected] = useState<Photo | null>(null);
  const [busySlot, setBusySlot] = useState<number | null>(null);

  const applyPhotos = (next: Photo[]) => {
    client.setQueryData<MyProfile>(queryKeys.myProfile, (data) => (data ? { ...data, photos: next } : data));
    void client.invalidateQueries({ queryKey: queryKeys.myProfile });
  };

  const onError = (error: unknown) => showToast(errorMessage(error), "error");

  const upload = useMutation({
    mutationFn: photosApi.upload,
    onSuccess: (photo) => applyPhotos([...photos, photo]),
    onError,
    onSettled: () => setBusySlot(null)
  });
  const replace = useMutation({
    mutationFn: ({ id, image }: { id: string; image: Parameters<typeof photosApi.replace>[1] }) => photosApi.replace(id, image),
    onSuccess: (photo) => applyPhotos(photos.map((p) => (p.id === photo.id ? photo : p))),
    onError,
    onSettled: () => setBusySlot(null)
  });
  const remove = useMutation({
    mutationFn: photosApi.remove,
    onSuccess: (_, id) => applyPhotos(photos.filter((p) => p.id !== id).map((p, i) => ({ ...p, position: i }))),
    onError,
    onSettled: () => setBusySlot(null)
  });
  const reorder = useMutation({
    mutationFn: photosApi.reorder,
    onSuccess: (data) => applyPhotos(data.photos),
    onError,
    onSettled: () => setBusySlot(null)
  });

  const addPhoto = async (slot: number) => {
    const image = await pickAndCompressPhoto();
    if (!image) return;
    setBusySlot(slot);
    upload.mutate(image);
  };

  const act = async (action: "main" | "replace" | "delete") => {
    const photo = selected;
    setSelected(null);
    if (!photo) return;
    if (action === "delete") {
      if (photos.length === 1) {
        showToast("Garde au moins une photo : ajoute-en une autre avant de supprimer celle-ci.", "error");
        return;
      }
      setBusySlot(photo.position);
      remove.mutate(photo.id);
    } else if (action === "main") {
      setBusySlot(0);
      reorder.mutate([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]);
    } else {
      const image = await pickAndCompressPhoto();
      if (!image) return;
      setBusySlot(photo.position);
      replace.mutate({ id: photo.id, image });
    }
  };

  return (
    <View>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }, (_, slot) => {
          const photo = photos[slot];
          const busy = busySlot === slot;
          const canAdd = !photo && slot === photos.length;
          return (
            <Pressable
              key={photo?.id ?? `empty-${slot}`}
              accessibilityRole="button"
              accessibilityLabel={photo ? `Photo ${slot + 1}${slot === 0 ? ", principale" : ""}. Options` : "Ajouter une photo"}
              disabled={busy || (!photo && !canAdd)}
              onPress={() => (photo ? setSelected(photo) : addPhoto(slot))}
              style={[styles.slot, !photo && styles.empty, !photo && !canAdd && styles.inactive]}
              testID={photo ? `photo-slot-${slot}` : canAdd ? "photo-add" : undefined}
            >
              {photo ? <Image source={{ uri: absoluteUrl(photo.url) }} style={StyleSheet.absoluteFill} contentFit="cover" transition={150} /> : null}
              {busy ? (
                <View style={styles.busy}>
                  <ActivityIndicator color={colors.primary} />
                </View>
              ) : !photo && canAdd ? (
                <Ionicons name="add" size={30} color={colors.primary} />
              ) : null}
              {photo && slot === 0 ? (
                <View style={styles.badge}>
                  <AppText variant="caption" color={colors.onPrimary}>
                    Principale
                  </AppText>
                </View>
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <AppText variant="caption" color={colors.textSubtle} style={styles.hint}>
        Jusqu’à 6 photos. Touche une photo pour la définir comme principale, la remplacer ou la supprimer.
      </AppText>
      <Sheet visible={!!selected} title="Cette photo" onClose={() => setSelected(null)}>
        {selected && selected.position !== 0 ? <Button label="Définir comme principale" icon="star-outline" variant="secondary" onPress={() => act("main")} /> : null}
        <Button label="Remplacer" icon="swap-horizontal-outline" variant="secondary" onPress={() => act("replace")} />
        <Button label="Supprimer" icon="trash-outline" variant="danger" onPress={() => act("delete")} testID="photo-delete" />
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: {
    width: "31%",
    aspectRatio: 4 / 5,
    borderRadius: radii.md,
    overflow: "hidden",
    backgroundColor: colors.surfaceRaised,
    alignItems: "center",
    justifyContent: "center"
  },
  empty: { borderWidth: 1.5, borderStyle: "dashed", borderColor: colors.borderStrong, backgroundColor: colors.surface },
  inactive: { opacity: 0.45 },
  busy: { ...StyleSheet.absoluteFillObject, backgroundColor: colors.scrim, alignItems: "center", justifyContent: "center" },
  badge: {
    position: "absolute",
    left: spacing.xs,
    bottom: spacing.xs,
    backgroundColor: colors.primary,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.xs,
    paddingVertical: 2
  },
  hint: { marginTop: spacing.sm }
});
