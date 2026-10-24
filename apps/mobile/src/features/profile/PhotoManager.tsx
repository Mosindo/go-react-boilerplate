import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { photoApi } from "../../api/endpoints";
import type { PhotoRef } from "../../api/types";
import { pickPhotoForm } from "../../lib/photoUpload";
import { errorMessage } from "../../lib/validation";
import { queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import { Icon, Photo, Text, useDialog, useToast } from "../../ui";

const MAX_PHOTOS = 6;

/** Grid of up to six photos: add, replace, delete and reorder (the first photo is the main one). */
export function PhotoManager({ photos }: { photos: PhotoRef[] }) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const dialog = useDialog();
  const toast = useToast();
  const [busy, setBusy] = useState(false);

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });

  const run = async (task: () => Promise<void>) => {
    setBusy(true);
    try {
      await task();
      await refresh();
    } catch (e) {
      toast.show({ message: errorMessage(e), kind: "error" });
    } finally {
      setBusy(false);
    }
  };

  const add = () =>
    run(async () => {
      const picked = await pickPhotoForm();
      if ("error" in picked) throw new Error(picked.error);
      if ("cancelled" in picked) return;
      await photoApi.upload(picked.form);
    });

  const move = (photo: PhotoRef, to: number) =>
    run(async () => {
      const ids = photos.map((p) => p.id);
      ids.splice(ids.indexOf(photo.id), 1);
      ids.splice(to, 0, photo.id);
      await photoApi.reorder(ids);
    });

  const openActions = async (photo: PhotoRef, index: number) => {
    const actions = [
      ...(index > 0
        ? [{ label: "Définir comme photo principale", value: "main", variant: "primary" as const }]
        : []),
      ...(index > 0
        ? [{ label: "Déplacer vers la gauche", value: "left", variant: "secondary" as const }]
        : []),
      ...(index < photos.length - 1
        ? [{ label: "Déplacer vers la droite", value: "right", variant: "secondary" as const }]
        : []),
      { label: "Remplacer", value: "replace", variant: "secondary" as const },
      { label: "Supprimer", value: "delete", variant: "danger" as const },
      { label: "Annuler", value: "cancel", variant: "ghost" as const },
    ];
    const choice = await dialog.choose({ title: "Photo", actions });
    switch (choice) {
      case "main":
        return move(photo, 0);
      case "left":
        return move(photo, index - 1);
      case "right":
        return move(photo, index + 1);
      case "replace":
        return run(async () => {
          const picked = await pickPhotoForm();
          if ("error" in picked) throw new Error(picked.error);
          if ("cancelled" in picked) return;
          await photoApi.replace(photo.id, picked.form);
        });
      case "delete":
        if (photos.length <= 1) {
          toast.show({ message: "Gardez au moins une photo pour être visible.", kind: "error" });
          return;
        }
        return run(() => photoApi.remove(photo.id));
    }
  };

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);

  return (
    <View style={styles.grid} testID="photo-grid">
      {slots.map((photo, index) => (
        <Pressable
          key={photo?.id ?? `empty-${index}`}
          testID={photo ? `photo-slot-${index}` : `photo-add-${index}`}
          accessibilityRole="button"
          accessibilityLabel={photo ? `Photo ${index + 1}, options` : "Ajouter une photo"}
          disabled={busy || (!photo && index !== photos.length)}
          onPress={() => (photo ? openActions(photo, index) : add())}
          style={[
            styles.slot,
            {
              backgroundColor: colors.surfaceAlt,
              borderColor: colors.border,
              opacity: busy ? 0.6 : 1,
            },
          ]}
        >
          {photo ? (
            <>
              <Photo photoId={photo.id} size="thumb" style={StyleSheet.absoluteFill} />
              {index === 0 ? (
                <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                  <Text variant="caption" color={colors.onPrimary}>
                    Principale
                  </Text>
                </View>
              ) : null}
            </>
          ) : (
            <Icon
              name="add"
              size={30}
              color={index === photos.length ? colors.primary : colors.textMuted}
            />
          )}
        </Pressable>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: {
    width: "31.5%",
    aspectRatio: 3 / 4,
    borderRadius: radius.md,
    borderWidth: 1,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center",
  },
  badge: {
    position: "absolute",
    left: 6,
    bottom: 6,
    borderRadius: radius.pill,
    paddingHorizontal: 8,
    paddingVertical: 2,
  },
});
