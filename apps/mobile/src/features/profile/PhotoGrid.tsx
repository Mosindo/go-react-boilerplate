import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { mediaUrl } from "../../api/client";
import { photosApi } from "../../api/endpoints";
import type { MyProfile, PhotoRef } from "../../api/types";
import { errorMessage } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { ActionSheet, Badge, showToast, Text } from "../../ui";
import { pickPhoto } from "./usePhotoPicker";

const SLOTS = 6;

export function PhotoGrid({ photos }: { photos: PhotoRef[] }) {
  const { colors, radii } = useTheme();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<PhotoRef | null>(null);
  const [busySlot, setBusySlot] = useState<number | null>(null);

  const applyPhotos = (next: PhotoRef[]) => {
    queryClient.setQueryData<MyProfile>(keys.profile, (p) => (p ? { ...p, photos: next } : p));
    void queryClient.invalidateQueries({ queryKey: keys.profile });
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
    onSuccess: (photo, vars) => applyPhotos(photos.map((p) => (p.id === vars.id ? photo : p))),
    onError,
    onSettled: () => setBusySlot(null)
  });
  const remove = useMutation({
    mutationFn: photosApi.remove,
    onSuccess: (_, id) => applyPhotos(photos.filter((p) => p.id !== id).map((p, i) => ({ ...p, position: i }))),
    onError
  });
  const reorder = useMutation({
    mutationFn: photosApi.reorder,
    onSuccess: (result) => applyPhotos(result.photos),
    onError
  });

  const add = async (slot: number) => {
    const image = await pickPhoto();
    if (image) {
      setBusySlot(slot);
      upload.mutate(image);
    }
  };

  return (
    <View style={styles.grid}>
      {Array.from({ length: SLOTS }, (_, slot) => {
        const photo = photos[slot];
        const busy = busySlot === slot;
        return (
          <Pressable
            accessibilityLabel={photo ? `Photo ${slot + 1}${slot === 0 ? ", principale" : ""}` : "Ajouter une photo"}
            accessibilityRole="button"
            disabled={busy || (!photo && slot !== photos.length)}
            key={photo?.id ?? `empty-${slot}`}
            onPress={() => (photo ? setSelected(photo) : void add(slot))}
            style={[
              styles.slot,
              { borderRadius: radii.md, backgroundColor: colors.surfaceAlt, borderColor: colors.border, opacity: !photo && slot > photos.length ? 0.5 : 1 }
            ]}
            testID={photo ? `photo-slot-${slot}` : `photo-add-${slot}`}
          >
            {photo ? (
              <Image cachePolicy="memory-disk" contentFit="cover" source={{ uri: mediaUrl(photo.url) }} style={StyleSheet.absoluteFill} transition={150} />
            ) : busy ? (
              <ActivityIndicator color={colors.primary} />
            ) : (
              <Ionicons color={slot === photos.length ? colors.primary : colors.textSubtle} name="add-circle" size={30} />
            )}
            {photo && slot === 0 ? (
              <View style={styles.badge}>
                <Badge label="Principale" />
              </View>
            ) : null}
          </Pressable>
        );
      })}
      <ActionSheet
        actions={
          selected
            ? [
                ...(selected.position > 0
                  ? [
                      {
                        label: "Définir comme photo principale",
                        onPress: () => reorder.mutate([selected.id, ...photos.filter((p) => p.id !== selected.id).map((p) => p.id)])
                      }
                    ]
                  : []),
                {
                  label: "Remplacer",
                  onPress: () =>
                    void pickPhoto().then((image) => {
                      if (image) {
                        setBusySlot(selected.position);
                        replace.mutate({ id: selected.id, image });
                      }
                    })
                },
                ...(photos.length > 1 ? [{ label: "Supprimer", destructive: true, onPress: () => remove.mutate(selected.id) }] : [])
              ]
            : []
        }
        message={photos.length <= 1 ? "Garde au moins une photo sur ton profil." : undefined}
        onClose={() => setSelected(null)}
        title="Photo"
        visible={!!selected}
      />
      <Text style={styles.help} tone="subtle" variant="caption">
        Jusqu’à 6 photos. La première est ta photo principale. Les métadonnées (dont la position GPS) sont supprimées.
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", justifyContent: "space-between", rowGap: 10 },
  slot: { width: "31.8%", aspectRatio: 3 / 4, borderWidth: 1, overflow: "hidden", alignItems: "center", justifyContent: "center" },
  badge: { position: "absolute", left: 6, bottom: 6 },
  help: { width: "100%" }
});
