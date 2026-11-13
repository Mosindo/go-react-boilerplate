import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { profileApi } from "../../api/profile";
import type { OwnProfile, PhotoView } from "../../api/types";
import { queryKeys } from "../../state/queryClient";
import { useTheme } from "../../theme/ThemeProvider";
import { Photo, Sheet, Text, useToast, type SheetAction } from "../../ui";
import { describeError } from "../../utils/errors";
import { pickAndPreparePhoto } from "../../utils/images";

const SLOTS = 6;

type Props = { photos: PhotoView[] };

/** 6-slot grid: add, replace, delete and reorder photos; slot 1 is the main photo. */
export function PhotoManager({ photos }: Props) {
  const { colors, radii } = useTheme();
  const toast = useToast();
  const client = useQueryClient();
  const [busy, setBusy] = useState<string | null>(null);
  const [selected, setSelected] = useState<PhotoView | null>(null);

  const setPhotos = (next: PhotoView[]) => {
    client.setQueryData<OwnProfile>(queryKeys.myProfile, (p) => (p ? { ...p, photos: next } : p));
    void client.invalidateQueries({ queryKey: queryKeys.myProfile });
  };

  const run = async (key: string, task: () => Promise<void>) => {
    setBusy(key);
    try {
      await task();
    } catch (error) {
      if (error instanceof Error && error.message === "permission") {
        toast.show("Autorisez l'accès à vos photos dans les réglages.", "error");
      } else {
        toast.show(describeError(error, "Impossible d'envoyer cette photo."), "error");
      }
    } finally {
      setBusy(null);
    }
  };

  const add = () =>
    run("new", async () => {
      const uri = await pickAndPreparePhoto();
      if (!uri) return;
      const photo = await profileApi.uploadPhoto(uri);
      setPhotos([...photos, photo]);
    });

  const replace = (photo: PhotoView) =>
    run(photo.id, async () => {
      const uri = await pickAndPreparePhoto();
      if (!uri) return;
      const updated = await profileApi.replacePhoto(photo.id, uri);
      setPhotos(photos.map((p) => (p.id === photo.id ? updated : p)));
    });

  const remove = (photo: PhotoView) =>
    run(photo.id, async () => {
      await profileApi.deletePhoto(photo.id);
      setPhotos(photos.filter((p) => p.id !== photo.id).map((p, i) => ({ ...p, position: i })));
    });

  const move = (photo: PhotoView, to: number) =>
    run(photo.id, async () => {
      const ids = photos.map((p) => p.id).filter((id) => id !== photo.id);
      ids.splice(to, 0, photo.id);
      setPhotos(await profileApi.reorderPhotos(ids));
    });

  const actionsFor = (photo: PhotoView): SheetAction[] => {
    const index = photos.findIndex((p) => p.id === photo.id);
    const actions: SheetAction[] = [];
    if (index > 0) {
      actions.push({ label: "Choisir comme photo principale", icon: "star-outline", onPress: () => move(photo, 0) });
      actions.push({ label: "Déplacer avant", icon: "arrow-back-outline", onPress: () => move(photo, index - 1) });
    }
    if (index < photos.length - 1) {
      actions.push({ label: "Déplacer après", icon: "arrow-forward-outline", onPress: () => move(photo, index + 1) });
    }
    actions.push({ label: "Remplacer", icon: "swap-horizontal-outline", onPress: () => replace(photo) });
    actions.push({
      label: "Supprimer",
      icon: "trash-outline",
      destructive: true,
      onPress: () => remove(photo),
      testID: "photo-delete"
    });
    return actions;
  };

  return (
    <View style={styles.grid}>
      {Array.from({ length: SLOTS }).map((_, i) => {
        const photo = photos[i];
        const isBusy = photo ? busy === photo.id : busy === "new" && i === photos.length;
        const isNextSlot = !photo && i === photos.length;
        return (
          <Pressable
            key={photo?.id ?? `slot-${i}`}
            testID={photo ? `photo-slot-${i}` : isNextSlot ? "photo-add" : undefined}
            accessibilityRole="button"
            accessibilityLabel={photo ? `Photo ${i + 1}${i === 0 ? ", principale" : ""}` : "Ajouter une photo"}
            disabled={!!busy || (!photo && !isNextSlot)}
            onPress={() => (photo ? setSelected(photo) : add())}
            style={[
              styles.slot,
              {
                borderRadius: radii.md,
                backgroundColor: colors.surfaceAlt,
                borderColor: isNextSlot ? colors.primary : colors.border,
                borderStyle: photo ? "solid" : "dashed"
              }
            ]}
          >
            {photo ? <Photo url={photo.url} style={[StyleSheet.absoluteFill, { borderRadius: radii.md }]} /> : null}
            {!photo ? <Ionicons name="add" size={28} color={isNextSlot ? colors.primary : colors.textSubtle} /> : null}
            {i === 0 && photo ? (
              <View style={[styles.mainTag, { backgroundColor: colors.primary }]}>
                <Text variant="caption" weight="semibold" tone="onPrimary">
                  Principale
                </Text>
              </View>
            ) : null}
            {isBusy ? (
              <View
                style={[
                  StyleSheet.absoluteFill,
                  styles.busy,
                  { backgroundColor: colors.overlay, borderRadius: radii.md }
                ]}
              >
                <ActivityIndicator color="#FFFFFF" />
              </View>
            ) : null}
          </Pressable>
        );
      })}
      <Sheet
        visible={!!selected}
        title="Photo"
        onClose={() => setSelected(null)}
        actions={selected ? actionsFor(selected) : []}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  slot: {
    width: "31.5%",
    aspectRatio: 4 / 5,
    borderWidth: 1.5,
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden"
  },
  mainTag: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 8, paddingVertical: 2, borderRadius: 8 },
  busy: { alignItems: "center", justifyContent: "center" }
});
