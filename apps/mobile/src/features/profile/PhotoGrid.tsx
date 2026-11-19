import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { photosApi } from "../../api/endpoints";
import type { OwnProfile, Photo } from "../../api/types";
import { pickPhoto, photoFormData } from "../../lib/images";
import { useSetProfile } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { ActionSheet, Icon, RemoteImage, Text, useFeedback, type SheetAction } from "../../ui";

const SLOTS = 6;

/** Six-slot photo manager: add, replace, set as main, move, delete. Position 0 is the main photo. */
export function PhotoGrid({ profile }: { profile: OwnProfile }) {
  const { colors } = useTheme();
  const { toast, confirm } = useFeedback();
  const setProfile = useSetProfile();
  const [selected, setSelected] = useState<Photo | null>(null);
  const photos = profile.photos;

  const update = (next: Photo[]) => setProfile({ ...profile, photos: next, isComplete: next.length > 0 });

  const upload = useMutation({
    mutationFn: async () => {
      const image = await pickPhoto();
      if (!image) return null;
      return photosApi.upload(await photoFormData(image));
    },
    onSuccess: (photo) => {
      if (photo) {
        update([...photos, photo]);
        toast("Photo ajoutée", "success");
      }
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const replace = useMutation({
    mutationFn: async (photo: Photo) => {
      const image = await pickPhoto();
      if (!image) return null;
      return photosApi.replace(photo.id, await photoFormData(image));
    },
    onSuccess: (next) => next && update(next),
    onError: (e) => toast(errorMessage(e), "error")
  });

  const reorder = useMutation({
    mutationFn: (ids: string[]) => photosApi.reorder(ids),
    onSuccess: update,
    onError: (e) => toast(errorMessage(e), "error")
  });

  const remove = useMutation({
    mutationFn: (photo: Photo) => photosApi.remove(photo.id),
    onSuccess: update,
    onError: (e) => toast(errorMessage(e), "error")
  });

  const move = (photo: Photo, to: number) => {
    const ids = photos.map((p) => p.id).filter((id) => id !== photo.id);
    ids.splice(to, 0, photo.id);
    reorder.mutate(ids);
  };

  const busy = upload.isPending || replace.isPending || reorder.isPending || remove.isPending;

  const actions: SheetAction[] = selected
    ? [
        ...(selected.position > 0
          ? [{ label: "Définir comme photo principale", icon: "star-outline" as const, onPress: () => move(selected, 0) }]
          : []),
        ...(selected.position > 0
          ? [{ label: "Déplacer vers la gauche", icon: "arrow-back" as const, onPress: () => move(selected, selected.position - 1) }]
          : []),
        ...(selected.position < photos.length - 1
          ? [{ label: "Déplacer vers la droite", icon: "arrow-forward" as const, onPress: () => move(selected, selected.position + 1) }]
          : []),
        { label: "Remplacer", icon: "swap-horizontal", onPress: () => replace.mutate(selected) },
        {
          label: "Supprimer",
          icon: "trash-outline",
          destructive: true,
          testID: "photo-delete",
          onPress: async () => {
            const ok = await confirm({ title: "Supprimer cette photo ?", confirmLabel: "Supprimer", destructive: true });
            if (ok) remove.mutate(selected);
          }
        }
      ]
    : [];

  return (
    <View>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }).map((_, i) => {
          const photo = photos[i];
          const isNext = i === photos.length;
          return (
            <Pressable
              key={photo?.id ?? `empty-${i}`}
              accessibilityRole="button"
              accessibilityLabel={photo ? `Photo ${i + 1}${i === 0 ? ", principale" : ""}. Ouvrir les options` : "Ajouter une photo"}
              disabled={busy || (!photo && !isNext)}
              onPress={() => (photo ? setSelected(photo) : upload.mutate())}
              testID={photo ? `photo-slot-${i}` : isNext ? "photo-add" : undefined}
              style={[styles.slot, { borderColor: colors.border, backgroundColor: colors.surfaceMuted }]}
            >
              {photo ? (
                <>
                  <RemoteImage path={photo.url} style={StyleSheet.absoluteFill} />
                  {i === 0 ? (
                    <View style={[styles.mainBadge, { backgroundColor: colors.primary }]}>
                      <Text variant="caption" tone="onPrimary">
                        Principale
                      </Text>
                    </View>
                  ) : null}
                </>
              ) : isNext ? (
                upload.isPending ? (
                  <ActivityIndicator color={colors.primary} />
                ) : (
                  <View style={[styles.plus, { backgroundColor: colors.primary }]}>
                    <Icon name="add" size={22} color={colors.onPrimary} />
                  </View>
                )
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <Text variant="caption" tone="subtle" style={styles.help}>
        JPEG ou PNG, 8 Mo max. Les métadonnées (dont la position GPS) sont supprimées automatiquement.
      </Text>
      <ActionSheet visible={!!selected} title="Photo" actions={actions} onClose={() => setSelected(null)} />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: {
    width: "31%",
    aspectRatio: 3 / 4,
    borderRadius: radii.md,
    borderWidth: 1,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  mainBadge: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 8, paddingVertical: 2, borderRadius: radii.pill },
  plus: { width: 36, height: 36, borderRadius: 18, alignItems: "center", justifyContent: "center" },
  help: { marginTop: spacing.sm }
});
