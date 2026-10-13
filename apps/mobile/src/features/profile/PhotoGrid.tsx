import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deletePhoto, replacePhoto, reorderPhotos, uploadPhoto } from "../../api/platform";
import { errorMessage } from "../../api/client";
import type { PhotoRef } from "../../api/types";
import { pickPhoto } from "../../lib/images";
import { queryKeys } from "../../lib/queryClient";
import { ActionSheet, Badge, Loader, Notice, PhotoImage, Text, radii, spacing, type SheetAction } from "../../shared/ui";
import { useTheme } from "../../theme";

export const MAX_PHOTOS = 6;

type PhotoGridProps = {
  photos: PhotoRef[];
};

/** Add, reorder (first = main), replace and delete photos. Changes are saved immediately. */
export function PhotoGrid({ photos }: PhotoGridProps) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<PhotoRef | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });

  const mutation = useMutation({
    mutationFn: async (task: () => Promise<"cancelled" | "denied" | "done">) => task(),
    onSuccess: async (result) => {
      if (result === "denied") {
        setMessage("Autorisez l'accès à vos photos dans les réglages de l'appareil.");
      } else if (result === "done") {
        setMessage(null);
      }
      await refresh();
    },
    onError: (error) => setMessage(errorMessage(error, "Action impossible pour le moment."))
  });

  const run = (task: () => Promise<"cancelled" | "denied" | "done">) => mutation.mutate(task);

  const add = () =>
    run(async () => {
      const picked = await pickPhoto();
      if (picked.status !== "ok") {
        return picked.status;
      }
      await uploadPhoto(picked.file);
      return "done";
    });

  const replace = (photo: PhotoRef) =>
    run(async () => {
      const picked = await pickPhoto();
      if (picked.status !== "ok") {
        return picked.status;
      }
      await replacePhoto(photo.id, picked.file);
      return "done";
    });

  const move = (photo: PhotoRef, target: number) =>
    run(async () => {
      const ids = photos.map((p) => p.id);
      const from = ids.indexOf(photo.id);
      ids.splice(from, 1);
      ids.splice(target, 0, photo.id);
      await reorderPhotos(ids);
      return "done";
    });

  const remove = (photo: PhotoRef) =>
    run(async () => {
      await deletePhoto(photo.id);
      return "done";
    });

  const actions: SheetAction[] = [];
  if (selected) {
    const index = photos.findIndex((p) => p.id === selected.id);
    if (index > 0) {
      actions.push({ label: "Définir comme photo principale", onPress: () => move(selected, 0) });
      actions.push({ label: "Déplacer vers la gauche", onPress: () => move(selected, index - 1) });
    }
    if (index >= 0 && index < photos.length - 1) {
      actions.push({ label: "Déplacer vers la droite", onPress: () => move(selected, index + 1) });
    }
    actions.push({ label: "Remplacer la photo", onPress: () => replace(selected) });
    actions.push({ label: "Supprimer", destructive: true, onPress: () => remove(selected) });
  }

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);
  return (
    <View style={styles.root}>
      {message ? <Notice message={message} tone="warning" /> : null}
      <View style={styles.grid}>
        {slots.map((photo, index) => (
          <View key={photo?.id ?? `empty-${index}`} style={styles.cell}>
            {photo ? (
              <Pressable
                accessibilityHint="Ouvre les actions de la photo"
                accessibilityLabel={index === 0 ? "Photo principale" : `Photo ${index + 1}`}
                accessibilityRole="button"
                onPress={() => setSelected(photo)}
                style={styles.fill}
                testID={`photo-${index}`}
              >
                <PhotoImage photoId={photo.id} size="thumb" style={styles.image} />
                {index === 0 ? <Badge label="Principale" style={styles.mainBadge} /> : null}
              </Pressable>
            ) : (
              <Pressable
                accessibilityLabel="Ajouter une photo"
                accessibilityRole="button"
                disabled={mutation.isPending}
                onPress={add}
                style={[styles.fill, styles.empty, { borderColor: colors.border, backgroundColor: colors.surface }]}
                testID={index === photos.length ? "photo-add" : undefined}
              >
                <Ionicons color={colors.textSubtle} name="add" size={28} />
              </Pressable>
            )}
          </View>
        ))}
      </View>
      {mutation.isPending ? (
        <View style={styles.pending}>
          <Loader />
          <Text tone="muted" variant="label">
            Envoi en cours…
          </Text>
        </View>
      ) : (
        <Text tone="subtle" variant="caption">
          {photos.length}/{MAX_PHOTOS} photos · la première est votre photo principale.
        </Text>
      )}
      <ActionSheet actions={actions} onClose={() => setSelected(null)} title="Cette photo" visible={selected !== null} />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 3 / 4 },
  fill: { flex: 1, borderRadius: radii.md, overflow: "hidden" },
  image: { width: "100%", height: "100%" },
  empty: { borderWidth: 1.5, borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  mainBadge: { position: "absolute", left: spacing.xs, bottom: spacing.xs },
  pending: { flexDirection: "row", alignItems: "center", gap: spacing.sm }
});
