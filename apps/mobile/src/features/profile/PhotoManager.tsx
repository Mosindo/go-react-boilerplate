import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { photosApi } from "../../api/endpoints";
import type { PhotoRef } from "../../api/types";
import { keys } from "../../hooks/queries";
import { pickPhoto } from "../../lib/device";
import { radii, spacing, useTheme } from "../../theme/theme";
import { ActionSheet } from "../../ui/Modal";
import { Photo } from "../../ui/Photo";
import { Text } from "../../ui/Text";
import { useToast } from "../../ui/Toast";

const MAX_PHOTOS = 6;

/** Grid of up to 6 photos: add, replace, delete and choose the main photo (position 0). */
export function PhotoManager({ photos }: { photos: PhotoRef[] }) {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<PhotoRef | null>(null);

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: keys.profile });
    void queryClient.invalidateQueries({ queryKey: keys.discover });
  };
  const fail = (e: unknown) => toast.show(e instanceof Error ? e.message : "Action impossible.", "error");
  const failUpload = (e: unknown) =>
    toast.show(e instanceof ApiError || e instanceof Error ? e.message : "Envoi impossible.", "error");

  const add = useMutation({
    mutationFn: async () => {
      const file = await pickPhoto();
      if (file) await photosApi.upload(file);
    },
    onSuccess: refresh,
    onError: failUpload
  });
  const replace = useMutation({
    mutationFn: async (id: string) => {
      const file = await pickPhoto();
      if (file) await photosApi.replace(id, file);
    },
    onSuccess: refresh,
    onError: failUpload
  });
  const remove = useMutation({ mutationFn: photosApi.remove, onSuccess: refresh, onError: fail });
  const reorder = useMutation({ mutationFn: photosApi.reorder, onSuccess: refresh, onError: fail });

  const makeMain = (photo: PhotoRef) =>
    reorder.mutate([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]);
  const busy = add.isPending || replace.isPending || remove.isPending || reorder.isPending;

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);
  const actions = selected
    ? [
        ...(selected.position > 0
          ? [{ label: "Définir comme photo principale", onPress: () => makeMain(selected) }]
          : []),
        { label: "Remplacer", onPress: () => replace.mutate(selected.id) },
        ...(photos.length > 1
          ? [{ label: "Supprimer", destructive: true, onPress: () => remove.mutate(selected.id) }]
          : [])
      ]
    : [];

  return (
    <View>
      <View style={styles.grid}>
        {slots.map((photo, i) => (
          <Pressable
            key={photo?.id ?? `empty-${i}`}
            accessibilityRole="button"
            accessibilityLabel={
              photo ? `Photo ${i + 1}${i === 0 ? ", principale" : ""}, modifier` : "Ajouter une photo"
            }
            disabled={busy}
            onPress={() => (photo ? setSelected(photo) : add.mutate())}
            style={[
              styles.slot,
              { backgroundColor: colors.surfaceMuted, borderColor: colors.border, opacity: busy ? 0.6 : 1 }
            ]}
            testID={photo ? `photo-${i}` : i === photos.length ? "photo-add" : `photo-empty-${i}`}
          >
            {photo ? (
              <>
                <Photo path={photo.url} style={StyleSheet.absoluteFill} />
                {i === 0 ? (
                  <View style={[styles.main, { backgroundColor: colors.primary }]}>
                    <Text variant="caption" tone="onPrimary">
                      Principale
                    </Text>
                  </View>
                ) : null}
              </>
            ) : (
              <Ionicons name="add" size={32} color={colors.textSubtle} />
            )}
          </Pressable>
        ))}
      </View>
      <Text variant="caption" tone="textSubtle" style={styles.hint}>
        JPEG ou PNG, 8 Mo maximum. Les métadonnées (position GPS, appareil) sont retirées automatiquement.
      </Text>
      <ActionSheet visible={!!selected} onClose={() => setSelected(null)} actions={actions} />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  slot: {
    width: "31.5%",
    aspectRatio: 3 / 4,
    borderRadius: radii.md,
    borderWidth: 1,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  main: {
    position: "absolute",
    left: 6,
    bottom: 6,
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 2
  },
  hint: { marginTop: spacing.xs }
});
