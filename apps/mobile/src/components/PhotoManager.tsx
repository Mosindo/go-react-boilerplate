import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deletePhoto, replacePhoto, reorderPhotos, uploadPhoto, type PickedImage } from "../api/profile";
import { errorMessage } from "../api/client";
import type { Photo } from "../api/types";
import { queryKeys } from "../hooks/queries";
import { showToast } from "../shared/feedback";
import { Button, Text, colors, radii, spacing } from "../shared/ui";
import { ConfirmModal } from "./ConfirmModal";
import { PhotoImage } from "./PhotoImage";
import { MAX_PHOTOS } from "./labels";

type PhotoManagerProps = {
  photos: Photo[];
};

async function pickImage(): Promise<PickedImage | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsMultipleSelection: false,
    quality: 0.85
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  return { uri: asset.uri, mimeType: asset.mimeType, fileName: asset.fileName };
}

export function PhotoManager({ photos }: PhotoManagerProps) {
  const client = useQueryClient();
  const [selected, setSelected] = useState<Photo | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const refresh = () => client.invalidateQueries({ queryKey: queryKeys.profile });
  const onError = (error: unknown) => showToast(errorMessage(error, "L'opération sur la photo a échoué."), "error");

  const add = useMutation({
    mutationFn: async () => {
      const image = await pickImage();
      return image ? uploadPhoto(image) : null;
    },
    onSuccess: refresh,
    onError
  });
  const replace = useMutation({
    mutationFn: async (photoId: string) => {
      const image = await pickImage();
      return image ? replacePhoto(photoId, image) : null;
    },
    onSuccess: refresh,
    onError
  });
  const remove = useMutation({ mutationFn: (photoId: string) => deletePhoto(photoId), onSuccess: refresh, onError });
  const makePrimary = useMutation({
    mutationFn: (photoId: string) => reorderPhotos([photoId, ...photos.filter((p) => p.id !== photoId).map((p) => p.id)]),
    onSuccess: refresh,
    onError
  });

  const busy = add.isPending || replace.isPending || remove.isPending || makePrimary.isPending;
  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);

  return (
    <View style={styles.wrap}>
      <View style={styles.grid}>
        {slots.map((photo, i) => (
          <View key={photo?.id ?? `empty-${i}`} style={styles.cell}>
            {photo ? (
              <Pressable
                accessibilityLabel={i === 0 ? "Photo principale, modifier" : `Photo ${i + 1}, modifier`}
                accessibilityRole="button"
                disabled={busy}
                onPress={() => setSelected(photo)}
                style={styles.slot}
                testID={`photo-slot-${i}`}
              >
                <PhotoImage path={photo.thumbUrl} style={StyleSheet.absoluteFill} />
                {i === 0 ? (
                  <View style={styles.badge}>
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
                disabled={busy || i > photos.length}
                onPress={() => add.mutate()}
                style={[styles.slot, styles.empty, i > photos.length ? styles.dim : null]}
                testID={i === photos.length ? "photo-add" : `photo-empty-${i}`}
              >
                <Text tone="primary" weight="bold" style={styles.plus}>
                  {add.isPending && i === photos.length ? "…" : "+"}
                </Text>
              </Pressable>
            )}
          </View>
        ))}
      </View>
      <Text tone="muted" variant="caption">
        Au moins une photo est nécessaire. JPEG ou PNG, 8 Mo maximum. La première photo est celle qui s'affiche en premier.
      </Text>

      <ConfirmModal
        cancelLabel="Fermer"
        confirmLabel={selected && selected.position !== 0 ? "Rendre principale" : "OK"}
        message="Que voulez-vous faire avec cette photo ?"
        onCancel={() => setSelected(null)}
        onConfirm={() => {
          if (selected && selected.position !== 0) {
            makePrimary.mutate(selected.id);
          }
          setSelected(null);
        }}
        title="Photo"
        visible={Boolean(selected) && !confirmDelete}
      >
        <View style={styles.sheetActions}>
          <Button
            label="Remplacer"
            onPress={() => {
              if (selected) replace.mutate(selected.id);
              setSelected(null);
            }}
            testID="photo-replace"
            variant="outline"
          />
          <Button label="Supprimer" onPress={() => setConfirmDelete(true)} testID="photo-delete" variant="destructive" />
        </View>
      </ConfirmModal>

      <ConfirmModal
        confirmLabel="Supprimer"
        destructive
        message="Cette photo sera supprimée définitivement."
        onCancel={() => setConfirmDelete(false)}
        onConfirm={() => {
          if (selected) remove.mutate(selected.id);
          setConfirmDelete(false);
          setSelected(null);
        }}
        title="Supprimer la photo ?"
        visible={confirmDelete}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 3 / 4 },
  slot: { flex: 1, borderRadius: radii.md, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  empty: { alignItems: "center", justifyContent: "center", borderWidth: 2, borderStyle: "dashed", borderColor: colors.primaryBorder, backgroundColor: colors.primarySoft },
  dim: { opacity: 0.4 },
  plus: { fontSize: 32, lineHeight: 36 },
  badge: { position: "absolute", left: 6, bottom: 6, backgroundColor: colors.scrim, borderRadius: radii.pill, paddingHorizontal: spacing.sm, paddingVertical: 2 },
  sheetActions: { gap: spacing.sm }
});
