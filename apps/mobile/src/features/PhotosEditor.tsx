import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { photoApi, type UploadFile } from "../api/endpoints";
import type { Photo } from "../api/types";
import { describeError } from "../lib/validation";
import { radii, useTheme } from "../theme/theme";
import { ActionSheet, ErrorState, LoadingView, PhotoImage, Text, useToast, type SheetAction } from "../ui";
import { keys } from "./keys";

const SLOTS = 6;

async function pickImage(): Promise<UploadFile | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Autorisez l'accès à vos photos dans les réglages.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 0.85
  });
  if (result.canceled || !result.assets[0]) {
    return null;
  }
  const asset = result.assets[0];
  const type = asset.mimeType === "image/png" ? "image/png" : "image/jpeg";
  return { uri: asset.uri, name: type === "image/png" ? "photo.png" : "photo.jpg", type };
}

export function PhotosEditor() {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<Photo | null>(null);
  const photos = useQuery({ queryKey: ["photos"], queryFn: photoApi.list });

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ["photos"] });
    void queryClient.invalidateQueries({ queryKey: keys.profile });
    void queryClient.invalidateQueries({ queryKey: keys.discover });
  };

  const run = useMutation({
    mutationFn: async (job: () => Promise<unknown>) => job(),
    onSuccess: refresh,
    onError: (error) => toast.show(describeError(error), "error")
  });

  const add = () =>
    run.mutate(async () => {
      const file = await pickImage();
      if (file) await photoApi.upload(file);
    });

  if (photos.isPending) return <LoadingView />;
  if (photos.isError) return <ErrorState message={describeError(photos.error)} onRetry={() => void photos.refetch()} />;

  const list = photos.data;
  const move = (photo: Photo, delta: number) => {
    const ids = list.map((p) => p.id);
    const from = ids.indexOf(photo.id);
    const to = from + delta;
    if (to < 0 || to >= ids.length) return;
    [ids[from], ids[to]] = [ids[to], ids[from]];
    run.mutate(() => photoApi.reorder(ids));
  };

  const actions: SheetAction[] = selected
    ? [
        ...(selected.position > 0
          ? [{ label: "Définir comme photo principale", onPress: () => run.mutate(() => photoApi.makePrimary(selected.id)) }]
          : []),
        ...(selected.position > 0 ? [{ label: "Déplacer vers la gauche", onPress: () => move(selected, -1) }] : []),
        ...(selected.position < list.length - 1 ? [{ label: "Déplacer vers la droite", onPress: () => move(selected, 1) }] : []),
        {
          label: "Remplacer",
          onPress: () =>
            run.mutate(async () => {
              const file = await pickImage();
              if (file) await photoApi.replace(selected.id, file);
            })
        },
        { label: "Supprimer", destructive: true, onPress: () => run.mutate(() => photoApi.remove(selected.id)) }
      ]
    : [];

  return (
    <View style={styles.wrap}>
      <Text variant="caption" muted>
        La première photo est votre photo principale. Appuyez sur une photo pour la gérer.
      </Text>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }, (_, index) => {
          const photo = list[index];
          return photo ? (
            <Pressable
              key={photo.id}
              accessibilityRole="button"
              accessibilityLabel={`Photo ${index + 1}`}
              onPress={() => setSelected(photo)}
              style={styles.slot}
            >
              <PhotoImage url={photo.url} style={styles.fill} />
              {index === 0 ? (
                <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                  <Text variant="caption" color={colors.onPrimary}>
                    Principale
                  </Text>
                </View>
              ) : null}
            </Pressable>
          ) : (
            <Pressable
              key={`empty-${index}`}
              accessibilityRole="button"
              accessibilityLabel="Ajouter une photo"
              disabled={run.isPending || index !== list.length}
              onPress={add}
              style={[styles.slot, styles.empty, { borderColor: colors.border, opacity: index === list.length ? 1 : 0.4 }]}
            >
              <Text variant="heading" muted>
                {run.isPending && index === list.length ? "…" : "+"}
              </Text>
            </Pressable>
          );
        })}
      </View>
      <ActionSheet visible={!!selected} actions={actions} onClose={() => setSelected(null)} />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 12 },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  slot: { width: "31%", aspectRatio: 3 / 4, borderRadius: radii.md, overflow: "hidden" },
  fill: { ...StyleSheet.absoluteFillObject },
  empty: { borderWidth: 2, borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  badge: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 8, paddingVertical: 2, borderRadius: radii.pill }
});
