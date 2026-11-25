import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { profileApi } from "../api/profile";
import type { Photo } from "../api/types";
import { queryKeys } from "../hooks/queryClient";
import { colors, radii, spacing } from "../theme";
import { Button, RemoteImage, Sheet, Text, showToast } from "../ui";
import { pickAndCompressPhoto } from "./imagePicking";

const SLOTS = 6;

type Op =
  { kind: "upload" } | { kind: "replace"; photoId: string } | { kind: "delete"; photoId: string } | { kind: "reorder"; ids: string[] };

export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const qc = useQueryClient();
  const [selected, setSelected] = useState<Photo | null>(null);
  const [busySlot, setBusySlot] = useState<number | null>(null);

  const mutation = useMutation({
    mutationFn: async (op: Op) => {
      if (op.kind === "upload" || op.kind === "replace") {
        const image = await pickAndCompressPhoto();
        if (!image) {
          return false;
        }
        if (op.kind === "upload") {
          await profileApi.uploadPhoto(image);
        } else {
          await profileApi.replacePhoto(op.photoId, image);
        }
        return true;
      }
      if (op.kind === "delete") {
        await profileApi.deletePhoto(op.photoId);
        return true;
      }
      await profileApi.reorderPhotos(op.ids);
      return true;
    },
    onSuccess: (changed) => {
      if (changed) {
        void qc.invalidateQueries({ queryKey: queryKeys.myProfile });
      }
    },
    onError: (error) => showToast(errorMessage(error, "Impossible de modifier la photo."), "error"),
    onSettled: () => setBusySlot(null)
  });

  const run = (op: Op, slot: number) => {
    setSelected(null);
    setBusySlot(slot);
    mutation.mutate(op);
  };

  const move = (photo: Photo, delta: number) => {
    const ids = photos.map((p) => p.id);
    const from = ids.indexOf(photo.id);
    const to = from + delta;
    if (to < 0 || to >= ids.length) {
      return;
    }
    [ids[from], ids[to]] = [ids[to], ids[from]];
    run({ kind: "reorder", ids }, to);
  };

  const makePrimary = (photo: Photo) => {
    const ids = [photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)];
    run({ kind: "reorder", ids }, 0);
  };

  return (
    <View>
      <View style={styles.grid} testID="photo-grid">
        {Array.from({ length: SLOTS }).map((_, slot) => {
          const photo = photos[slot];
          const busy = busySlot === slot && mutation.isPending;
          return (
            <Pressable
              key={photo?.id ?? `empty-${slot}`}
              accessibilityRole="button"
              accessibilityLabel={photo ? `Photo ${slot + 1}, options` : "Ajouter une photo"}
              disabled={mutation.isPending || (!photo && slot !== photos.length)}
              onPress={() => (photo ? setSelected(photo) : run({ kind: "upload" }, slot))}
              style={[styles.slot, !photo && styles.empty, !photo && slot !== photos.length && styles.inactive]}
              testID={photo ? `photo-slot-${slot}` : slot === photos.length ? "photo-add" : undefined}
            >
              {photo ? <RemoteImage path={photo.url} style={StyleSheet.absoluteFillObject} /> : null}
              {photo && slot === 0 ? (
                <View style={styles.primaryBadge}>
                  <Text variant="caption" color="onPrimary">
                    Principale
                  </Text>
                </View>
              ) : null}
              {!photo && slot === photos.length && !busy ? <Ionicons name="add" size={28} color={colors.primary} /> : null}
              {busy ? (
                <View style={styles.busy}>
                  <ActivityIndicator color={colors.primary} />
                </View>
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <Text variant="caption" color="textSubtle" style={styles.hint}>
        Jusqu’à 6 photos. La première est votre photo principale. Les métadonnées (dont la position GPS) sont supprimées.
      </Text>

      <Sheet visible={!!selected} title="Cette photo" onClose={() => setSelected(null)}>
        {selected ? (
          <View style={styles.actions}>
            {selected.position > 0 ? (
              <Button label="Définir comme principale" variant="secondary" icon="star-outline" onPress={() => makePrimary(selected)} />
            ) : null}
            <View style={styles.row}>
              <Button
                label="Avant"
                variant="secondary"
                icon="arrow-back"
                compact
                disabled={selected.position === 0}
                onPress={() => move(selected, -1)}
                style={styles.flex}
              />
              <Button
                label="Après"
                variant="secondary"
                icon="arrow-forward"
                compact
                disabled={selected.position === photos.length - 1}
                onPress={() => move(selected, 1)}
                style={styles.flex}
              />
            </View>
            <Button
              label="Remplacer"
              variant="secondary"
              icon="swap-horizontal"
              onPress={() => run({ kind: "replace", photoId: selected.id }, selected.position)}
            />
            <Button
              label="Supprimer"
              variant="danger"
              icon="trash-outline"
              disabled={photos.length <= 1}
              onPress={() => run({ kind: "delete", photoId: selected.id }, selected.position)}
              testID="photo-delete"
            />
            {photos.length <= 1 ? (
              <Text variant="caption" color="textSubtle" align="center">
                Gardez au moins une photo sur votre profil.
              </Text>
            ) : null}
          </View>
        ) : null}
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  },
  slot: {
    width: "31%",
    aspectRatio: 4 / 5,
    borderRadius: radii.md,
    overflow: "hidden",
    backgroundColor: colors.surface,
    alignItems: "center",
    justifyContent: "center"
  },
  empty: {
    borderWidth: 1.5,
    borderStyle: "dashed",
    borderColor: colors.borderStrong
  },
  inactive: {
    opacity: 0.45
  },
  primaryBadge: {
    position: "absolute",
    left: 6,
    bottom: 6,
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: radii.pill,
    backgroundColor: colors.primary
  },
  busy: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: colors.scrim,
    alignItems: "center",
    justifyContent: "center"
  },
  hint: {
    marginTop: spacing.sm
  },
  actions: {
    gap: spacing.sm
  },
  row: {
    flexDirection: "row",
    gap: spacing.sm
  },
  flex: {
    flex: 1
  }
});
