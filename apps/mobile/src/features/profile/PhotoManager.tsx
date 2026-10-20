import React, { useState } from "react";
import { Alert, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation } from "@tanstack/react-query";
import { deletePhoto, reorderPhotos, uploadPhoto } from "../../api/profile";
import type { Photo } from "../../api/types";
import { radii, spacing, useTheme } from "../../theme";
import { Button, RemoteImage, Text, useToast } from "../../ui";

export const MAX_PHOTOS = 6;

async function pickImage(): Promise<ImagePicker.ImagePickerAsset | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Allow photo access in your settings to add pictures.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.85
  });
  return result.canceled ? null : (result.assets[0] ?? null);
}

export function PhotoManager({
  photos,
  onChanged
}: {
  photos: Photo[];
  onChanged: () => Promise<void> | void;
}) {
  const { colors } = useTheme();
  const toast = useToast();
  const [busyId, setBusyId] = useState<string | null>(null);
  const ordered = [...photos].sort((a, b) => a.position - b.position);

  const run = async (id: string, action: () => Promise<unknown>, success?: string) => {
    setBusyId(id);
    try {
      await action();
      await onChanged();
      if (success) toast.show(success);
    } catch (error) {
      toast.show(error instanceof Error ? error.message : "Could not update photos.", "error");
    } finally {
      setBusyId(null);
    }
  };

  const add = useMutation({
    mutationFn: async () => {
      const asset = await pickImage();
      if (!asset) return;
      await uploadPhoto(asset.uri, asset.mimeType);
    },
    onSuccess: () => onChanged(),
    onError: (error) => toast.show(error instanceof Error ? error.message : "Upload failed.", "error")
  });

  const makeMain = (photo: Photo) =>
    run(
      photo.id,
      () => reorderPhotos([photo.id, ...ordered.filter((p) => p.id !== photo.id).map((p) => p.id)]),
      "Main photo updated"
    );

  const replace = (photo: Photo) =>
    run(photo.id, async () => {
      const asset = await pickImage();
      if (!asset) return;
      // At the photo limit the old picture must go first to make room for the new one.
      const atLimit = ordered.length >= MAX_PHOTOS;
      if (atLimit) await deletePhoto(photo.id);
      const uploaded = await uploadPhoto(asset.uri, asset.mimeType);
      const next = ordered.map((p) => (p.id === photo.id ? uploaded.id : p.id));
      if (atLimit) {
        await reorderPhotos(next);
      } else {
        await reorderPhotos([...next, photo.id]);
        await deletePhoto(photo.id);
      }
    });

  const remove = (photo: Photo) =>
    Alert.alert("Remove this photo?", undefined, [
      { text: "Cancel", style: "cancel" },
      { text: "Remove", style: "destructive", onPress: () => void run(photo.id, () => deletePhoto(photo.id)) }
    ]);

  return (
    <View>
      <View style={styles.grid}>
        {ordered.map((photo, index) => (
          <View
            key={photo.id}
            style={[styles.cell, { backgroundColor: colors.surface, borderColor: colors.border }]}
          >
            <RemoteImage url={photo.url} style={styles.image} accessibilityLabel={`Photo ${index + 1}`} />
            {index === 0 ? (
              <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                <Text variant="caption" tone="onPrimary">
                  Main
                </Text>
              </View>
            ) : null}
            <View style={styles.actions}>
              {index > 0 ? (
                <Button
                  label="Main"
                  variant="ghost"
                  onPress={() => void makeMain(photo)}
                  disabled={busyId !== null}
                  accessibilityLabel={`Make photo ${index + 1} your main photo`}
                />
              ) : null}
              <Button
                label="Replace"
                variant="ghost"
                onPress={() => void replace(photo)}
                disabled={busyId !== null}
                accessibilityLabel={`Replace photo ${index + 1}`}
              />
              <Button
                label="Remove"
                variant="danger"
                onPress={() => remove(photo)}
                disabled={busyId !== null}
                accessibilityLabel={`Remove photo ${index + 1}`}
              />
            </View>
          </View>
        ))}
      </View>
      {ordered.length < MAX_PHOTOS ? (
        <Button
          testID="photo-add-button"
          label={ordered.length === 0 ? "Add your first photo" : "Add a photo"}
          variant="secondary"
          loading={add.isPending}
          onPress={() => add.mutate()}
        />
      ) : null}
      <Text variant="caption" tone="muted" style={{ marginTop: spacing.sm }}>
        {ordered.length}/{MAX_PHOTOS} photos. The first one is shown first to everyone.
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { marginBottom: spacing.md },
  cell: { borderWidth: 1, borderRadius: radii.lg, overflow: "hidden", marginBottom: spacing.md },
  image: { width: "100%", aspectRatio: 4 / 5 },
  badge: {
    position: "absolute",
    top: spacing.md,
    left: spacing.md,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  actions: { flexDirection: "row", justifyContent: "space-around", paddingVertical: spacing.xs }
});
