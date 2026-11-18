import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { resolveMediaUrl } from "../api/client";
import { errorMessage } from "../api/errors";
import type { Photo } from "../api/types";
import { afterModalDismiss, Sheet, useFeedback } from "../feedback";
import { useDeletePhoto, useReorderPhotos, useReplacePhoto, useUploadPhoto } from "../hooks/queries";
import { radii, spacing, useTheme } from "../theme";
import { Icon, ListRow, Text } from "../ui";
import { pickPhoto } from "../utils/media";

const SLOTS = 6;

/** Six-slot photo manager: add, replace, delete and pick the main photo. */
export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const { colors } = useTheme();
  const { toast, confirm } = useFeedback();
  const upload = useUploadPhoto();
  const replace = useReplacePhoto();
  const remove = useDeletePhoto();
  const reorder = useReorderPhotos();
  const [selected, setSelected] = useState<Photo | null>(null);
  const busy = upload.isPending || replace.isPending || remove.isPending || reorder.isPending;

  const choose = async (): Promise<string | null> => {
    const result = await pickPhoto();
    if (result.status === "denied") {
      toast("Autorisez l’accès à vos photos dans les réglages.", "error");
      return null;
    }
    return result.status === "picked" ? result.uri : null;
  };

  const add = async () => {
    const uri = await choose();
    if (!uri) {
      return;
    }
    try {
      await upload.mutateAsync(uri);
      toast("Photo ajoutée", "success");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const run = async (action: () => Promise<unknown>, success: string) => {
    setSelected(null);
    try {
      await action();
      toast(success, "success");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const makeMain = (photo: Photo) =>
    run(() => reorder.mutateAsync([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]), "Photo principale mise à jour");

  const move = (photo: Photo, delta: number) => {
    const ids = photos.map((p) => p.id);
    const from = ids.indexOf(photo.id);
    const to = from + delta;
    if (to < 0 || to >= ids.length) {
      return;
    }
    [ids[from], ids[to]] = [ids[to], ids[from]];
    void run(() => reorder.mutateAsync(ids), "Ordre mis à jour");
  };

  const replaceSelected = async (photo: Photo) => {
    setSelected(null);
    await afterModalDismiss();
    const uri = await choose();
    if (uri) {
      await run(() => replace.mutateAsync({ photoId: photo.id, uri }), "Photo remplacée");
    }
  };

  const deleteSelected = async (photo: Photo) => {
    setSelected(null);
    await afterModalDismiss();
    const ok = await confirm({
      title: "Supprimer cette photo ?",
      message: photos.length === 1 ? "Sans photo, votre profil ne sera plus visible dans la découverte." : undefined,
      confirmLabel: "Supprimer",
      destructive: true
    });
    if (ok) {
      await run(() => remove.mutateAsync(photo.id), "Photo supprimée");
    }
  };

  return (
    <View style={styles.grid} testID="photo-grid">
      {Array.from({ length: SLOTS }).map((_, index) => {
        const photo = photos[index];
        if (photo) {
          return (
            <Pressable
              key={photo.id}
              accessibilityHint="Ouvre les options de la photo"
              accessibilityLabel={index === 0 ? "Photo principale" : `Photo ${index + 1}`}
              accessibilityRole="button"
              onPress={() => setSelected(photo)}
              style={[styles.slot, { backgroundColor: colors.surfaceAlt }]}
              testID={`photo-slot-${index}`}
            >
              <Image
                cachePolicy="memory-disk"
                contentFit="cover"
                recyclingKey={photo.id}
                source={{ uri: resolveMediaUrl(photo.url) }}
                style={styles.fill}
                transition={150}
              />
              {index === 0 ? (
                <View style={[styles.badge, { backgroundColor: colors.accent }]}>
                  <Text color="textOnAccent" variant="caption">
                    Principale
                  </Text>
                </View>
              ) : null}
            </Pressable>
          );
        }
        const isNext = index === photos.length;
        return (
          <Pressable
            key={`empty-${index}`}
            accessibilityLabel="Ajouter une photo"
            accessibilityRole="button"
            disabled={!isNext || busy}
            onPress={add}
            style={[styles.slot, styles.empty, { borderColor: isNext ? colors.accent : colors.border, backgroundColor: colors.surface }]}
            testID={isNext ? "photo-add" : undefined}
          >
            {isNext && upload.isPending ? (
              <ActivityIndicator color={colors.accent} />
            ) : (
              <Icon color={isNext ? "accent" : "textSubtle"} name="add" size={28} />
            )}
          </Pressable>
        );
      })}

      <Sheet onClose={() => setSelected(null)} title="Photo" visible={!!selected}>
        {selected ? (
          <>
            {photos[0]?.id !== selected.id ? (
              <ListRow icon="star-outline" onPress={() => makeMain(selected)} title="Définir comme photo principale" />
            ) : null}
            {photos.indexOf(selected) > 0 ? <ListRow icon="arrow-back" onPress={() => move(selected, -1)} title="Déplacer avant" /> : null}
            {photos.indexOf(selected) < photos.length - 1 ? (
              <ListRow icon="arrow-forward" onPress={() => move(selected, 1)} title="Déplacer après" />
            ) : null}
            <ListRow icon="swap-horizontal" onPress={() => replaceSelected(selected)} title="Remplacer" />
            <ListRow icon="trash-outline" onPress={() => deleteSelected(selected)} title="Supprimer" tone="danger" />
          </>
        ) : null}
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: { width: "31%", aspectRatio: 4 / 5, borderRadius: radii.md, overflow: "hidden", alignItems: "center", justifyContent: "center" },
  empty: { borderWidth: 1.5, borderStyle: "dashed" },
  fill: { width: "100%", height: "100%" },
  badge: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 8, paddingVertical: 2, borderRadius: 999 }
});
