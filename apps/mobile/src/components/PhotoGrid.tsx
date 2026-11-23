import React, { useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { photosApi } from "../api/dating";
import type { MyProfile, Photo as PhotoType } from "../api/types";
import { keys } from "../hooks/queries";
import { ConfirmDialog, showToast } from "../shared/feedback";
import { Button, Loader, Photo, Text, colors, radii, spacing } from "../shared/ui";
import { errorMessage } from "../utils/errors";
import { pickPhoto } from "../utils/media";

type PhotoGridProps = {
  photos: PhotoType[];
  maxPhotos: number;
};

type Action = { kind: "upload" } | { kind: "replace"; photoId: string };

export function PhotoGrid({ maxPhotos, photos }: PhotoGridProps) {
  const client = useQueryClient();
  const [selected, setSelected] = useState<PhotoType | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const applyPhotos = (next: PhotoType[]) => {
    client.setQueryData<MyProfile>(keys.profile, (profile) => {
      if (!profile) {
        return profile;
      }
      const missing =
        next.length === 0 ? Array.from(new Set([...profile.missing, "photo" as const])) : profile.missing.filter((m) => m !== "photo");
      return { ...profile, photos: next, missing, isComplete: missing.length === 0 };
    });
    void client.invalidateQueries({ queryKey: keys.profile });
  };

  const upload = useMutation({
    mutationFn: async (action: Action) => {
      const picked = await pickPhoto();
      if (!picked) {
        return null;
      }
      if ("error" in picked) {
        throw new Error(picked.error);
      }
      return action.kind === "upload" ? photosApi.upload(picked.uri) : photosApi.replace(action.photoId, picked.uri);
    },
    onMutate: () => setError(null),
    onSuccess: (res, action) => {
      if (res) {
        applyPhotos(res.photos);
        showToast(action.kind === "upload" ? "Photo ajoutée" : "Photo remplacée", "success");
      }
    },
    onError: (e) => setError(e instanceof Error && !("code" in e) ? e.message : errorMessage(e))
  });

  const remove = useMutation({
    mutationFn: (photoId: string) => photosApi.remove(photoId),
    onSuccess: (res) => {
      applyPhotos(res.photos);
      showToast("Photo supprimée");
    },
    onError: (e) => setError(errorMessage(e))
  });

  const reorder = useMutation({
    mutationFn: (ids: string[]) => photosApi.reorder(ids),
    onSuccess: (res) => applyPhotos(res.photos),
    onError: (e) => setError(errorMessage(e))
  });

  const move = (photo: PhotoType, to: number) => {
    const ids = photos.map((p) => p.id).filter((id) => id !== photo.id);
    ids.splice(to, 0, photo.id);
    reorder.mutate(ids);
    setSelected(null);
  };

  const busy = upload.isPending || remove.isPending || reorder.isPending;
  const slots = Array.from({ length: maxPhotos }, (_, i) => photos[i] ?? null);

  return (
    <View style={styles.wrap}>
      <View style={styles.grid}>
        {slots.map((photo, index) => (
          <Pressable
            accessibilityLabel={photo ? `Photo ${index + 1}, options` : "Ajouter une photo"}
            accessibilityRole="button"
            disabled={busy || (!photo && index !== photos.length)}
            key={photo?.id ?? `empty-${index}`}
            onPress={() => (photo ? setSelected(photo) : upload.mutate({ kind: "upload" }))}
            style={({ pressed }) => [styles.slot, pressed ? styles.pressed : null]}
            testID={photo ? `photo-slot-${index}` : index === photos.length ? "photo-add" : undefined}
          >
            {photo ? (
              <>
                <Photo style={styles.image} url={photo.url} />
                {index === 0 ? (
                  <View style={styles.primaryTag}>
                    <Text tone="onPrimary" variant="caption" weight="bold">
                      Principale
                    </Text>
                  </View>
                ) : null}
              </>
            ) : (
              <View style={[styles.empty, index === photos.length ? styles.emptyNext : null]}>
                {busy && index === photos.length ? (
                  <Loader />
                ) : (
                  <Text tone={index === photos.length ? "primary" : "subtle"} variant="heading">
                    +
                  </Text>
                )}
              </View>
            )}
          </Pressable>
        ))}
      </View>
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="label">
          {error}
        </Text>
      ) : (
        <Text tone="subtle" variant="caption">
          {photos.length}/{maxPhotos} photos · la première est ta photo principale. Les métadonnées (dont la position GPS) sont supprimées.
        </Text>
      )}

      <Modal animationType="fade" onRequestClose={() => setSelected(null)} transparent visible={selected !== null && !confirmDelete}>
        <Pressable onPress={() => setSelected(null)} style={styles.backdrop}>
          <Pressable style={styles.sheet}>
            <Text variant="heading">Photo</Text>
            {selected && photos.findIndex((p) => p.id === selected.id) > 0 ? (
              <>
                <Button label="Définir comme principale" onPress={() => move(selected, 0)} variant="secondary" />
                <Button
                  label="Avancer d'une place"
                  onPress={() => move(selected, photos.findIndex((p) => p.id === selected.id) - 1)}
                  variant="secondary"
                />
              </>
            ) : null}
            <Button
              label="Remplacer"
              onPress={() => {
                if (selected) {
                  upload.mutate({ kind: "replace", photoId: selected.id });
                }
                setSelected(null);
              }}
              variant="secondary"
            />
            <Button
              disabled={photos.length <= 1}
              label={photos.length <= 1 ? "Garde au moins une photo" : "Supprimer"}
              onPress={() => setConfirmDelete(true)}
              variant="danger"
            />
            <Button label="Fermer" onPress={() => setSelected(null)} variant="ghost" />
          </Pressable>
        </Pressable>
      </Modal>
      <ConfirmDialog
        confirmLabel="Supprimer"
        destructive
        message="Cette photo sera définitivement supprimée."
        onCancel={() => setConfirmDelete(false)}
        onConfirm={() => {
          if (selected) {
            remove.mutate(selected.id);
          }
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
  wrap: {
    gap: spacing.md
  },
  grid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  },
  slot: {
    width: "31.5%",
    aspectRatio: 4 / 5,
    borderRadius: radii.md,
    overflow: "hidden"
  },
  pressed: {
    opacity: 0.8
  },
  image: {
    width: "100%",
    height: "100%"
  },
  empty: {
    flex: 1,
    borderRadius: radii.md,
    borderWidth: 1,
    borderStyle: "dashed",
    borderColor: colors.border,
    backgroundColor: colors.surfaceMuted,
    alignItems: "center",
    justifyContent: "center"
  },
  emptyNext: {
    borderColor: colors.primaryBorder,
    backgroundColor: colors.primarySoft
  },
  primaryTag: {
    position: "absolute",
    left: spacing.xs,
    bottom: spacing.xs,
    backgroundColor: colors.primary,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2
  },
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay,
    justifyContent: "flex-end"
  },
  sheet: {
    width: "100%",
    maxWidth: 640,
    alignSelf: "center",
    backgroundColor: colors.backgroundElevated,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    paddingBottom: spacing.xxxl,
    gap: spacing.sm
  }
});
