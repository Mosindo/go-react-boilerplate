import React, { useCallback, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { deletePhoto, errorMessage, getOwnProfile, queryKeys, reorderPhotos, replacePhoto, uploadPhoto, type LocalImage, type Photo } from "../api/platform";
import { Notice, Text, colors, radii, spacing } from "../shared/ui";
import { Button } from "../shared/ui";
import { PhotoImage } from "./PhotoImage";
import { showDialog } from "../shared/dialog";

const MAX_PHOTOS = 6;

async function pickImage(): Promise<LocalImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    showDialog("Photos are off", "Allow photo access in your device settings to add pictures.");
    return null;
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.85
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  return { uri: asset.uri, fileName: asset.fileName, mimeType: asset.mimeType };
}

/** Grid of up to six photos: add, replace, delete, and move to reorder (first = main photo). */
export function PhotoManager() {
  const client = useQueryClient();
  const profile = useQuery({ queryKey: queryKeys.profile, queryFn: getOwnProfile });
  const [error, setError] = useState<string | null>(null);
  const photos: Photo[] = profile.data?.photos ?? [];

  const refresh = useCallback(() => client.invalidateQueries({ queryKey: queryKeys.profile }), [client]);
  const onError = (e: unknown) => setError(errorMessage(e));

  const add = useMutation({ mutationFn: uploadPhoto, onSuccess: refresh, onError });
  const replace = useMutation({ mutationFn: (v: { id: string; image: LocalImage }) => replacePhoto(v.id, v.image), onSuccess: refresh, onError });
  const remove = useMutation({ mutationFn: deletePhoto, onSuccess: refresh, onError });
  const reorder = useMutation({ mutationFn: reorderPhotos, onSuccess: refresh, onError });
  const busy = add.isPending || replace.isPending || remove.isPending || reorder.isPending;

  const addPhoto = async () => {
    setError(null);
    const image = await pickImage();
    if (image) {
      add.mutate(image);
    }
  };

  const open = (photo: Photo, index: number) => {
    const move = (to: number) => {
      const ids = photos.map((p) => p.id);
      ids.splice(to, 0, ids.splice(index, 1)[0]);
      reorder.mutate(ids);
    };
    const buttons = [
      ...(index > 0 ? [{ text: index === 1 ? "Make main photo" : "Move earlier", onPress: () => move(index - 1) }] : []),
      ...(index < photos.length - 1 ? [{ text: "Move later", onPress: () => move(index + 1) }] : []),
      {
        text: "Replace",
        onPress: async () => {
          const image = await pickImage();
          if (image) {
            setError(null);
            replace.mutate({ id: photo.id, image });
          }
        }
      },
      {
        text: "Delete",
        style: "destructive" as const,
        onPress: () => {
          setError(null);
          remove.mutate(photo.id);
        }
      },
      { text: "Cancel", style: "cancel" as const }
    ];
    showDialog(index === 0 ? "Main photo" : `Photo ${index + 1}`, undefined, buttons);
  };

  return (
    <View style={styles.stack}>
      {error ? <Notice description={error} title="Photo problem" tone="danger" /> : null}
      <View style={styles.grid}>
        {photos.map((photo, index) => (
          <Pressable
            accessibilityHint="Opens options to move, replace or delete"
            accessibilityLabel={index === 0 ? "Main photo" : `Photo ${index + 1}`}
            accessibilityRole="button"
            disabled={busy}
            key={photo.id}
            onPress={() => open(photo, index)}
            style={styles.cell}
            testID={`photo-${index}`}
          >
            <PhotoImage path={photo.url} style={styles.image} />
            {index === 0 ? (
              <View style={styles.mainBadge}>
                <Text tone="inverse" variant="caption" weight="bold">
                  MAIN
                </Text>
              </View>
            ) : null}
          </Pressable>
        ))}
        {photos.length < MAX_PHOTOS ? (
          <Pressable
            accessibilityLabel="Add a photo"
            accessibilityRole="button"
            disabled={busy}
            onPress={() => void addPhoto()}
            style={[styles.cell, styles.add]}
            testID="photo-add"
          >
            <Text tone="primary" variant="title" weight="bold">
              +
            </Text>
          </Pressable>
        ) : null}
      </View>
      <Text tone="muted" variant="caption">
        {photos.length}/{MAX_PHOTOS} photos. JPEG or PNG up to 8 MB. The first photo is the one people see first.
      </Text>
      {busy ? <Button disabled label="Working…" loading size="sm" variant="ghost" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  stack: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: {
    width: "31.5%",
    aspectRatio: 0.8,
    borderRadius: radii.md,
    overflow: "hidden",
    backgroundColor: colors.surfaceSubtle
  },
  image: { width: "100%", height: "100%" },
  add: { alignItems: "center", justifyContent: "center", borderWidth: 1.5, borderStyle: "dashed", borderColor: colors.primaryBorder, backgroundColor: colors.surfaceAccent },
  mainBadge: { position: "absolute", left: spacing.xs, top: spacing.xs, backgroundColor: colors.scrim, borderRadius: radii.pill, paddingHorizontal: spacing.sm, paddingVertical: 2 }
});
