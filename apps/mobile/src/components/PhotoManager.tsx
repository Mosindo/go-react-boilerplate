import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import { Image } from "expo-image";
import * as ImagePicker from "expo-image-picker";
import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { absoluteUrl, errorMessage } from "../api/client";
import { datingApi, type PhotoFile } from "../api/dating";
import type { Photo } from "../api/types";
import { keys } from "../hooks/queries";
import { radii, spacing, useTheme } from "../theme";
import { Text, useToast } from "../ui";
import { ActionMenu } from "./ActionMenu";

const MAX_PHOTOS = 6;

async function pickPhoto(): Promise<PhotoFile | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.8
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  return { uri: asset.uri, name: asset.fileName ?? "photo.jpg", mimeType: asset.mimeType ?? "image/jpeg" };
}

/** Grid of up to six slots: add, replace, delete and choose the main photo (first slot). */
export function PhotoManager({ photos }: { photos: Photo[] }) {
  const { colors } = useTheme();
  const toast = useToast();
  const client = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<Photo | null>(null);

  const run = async (task: () => Promise<void>, success?: string) => {
    setBusy(true);
    try {
      await task();
      await client.invalidateQueries({ queryKey: keys.me });
      void client.invalidateQueries({ queryKey: keys.discover });
      if (success) {
        toast.show(success, "success");
      }
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(false);
    }
  };

  const add = async () => {
    const file = await pickPhoto();
    if (file) {
      await run(async () => void (await datingApi.uploadPhoto(file)), "Photo ajoutée");
    }
  };

  const replace = async (photo: Photo) => {
    const file = await pickPhoto();
    if (file) {
      await run(() => datingApi.replacePhoto(photo.id, file), "Photo remplacée");
    }
  };

  const makeMain = (photo: Photo) =>
    run(
      () => datingApi.reorderPhotos([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]),
      "Photo principale mise à jour"
    );

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);

  return (
    <View>
      <View style={styles.grid}>
        {slots.map((photo, index) => (
          <Pressable
            key={photo?.id ?? `empty-${index}`}
            accessibilityRole="button"
            accessibilityLabel={photo ? `Photo ${index + 1}, options` : "Ajouter une photo"}
            disabled={busy}
            onPress={() => (photo ? setSelected(photo) : void add())}
            style={[styles.slot, { backgroundColor: colors.surfaceMuted, borderColor: colors.border }]}
          >
            {photo ? (
              <>
                <Image
                  source={{ uri: absoluteUrl(photo.thumbUrl) }}
                  style={StyleSheet.absoluteFill}
                  contentFit="cover"
                  cachePolicy="memory-disk"
                />
                {index === 0 ? (
                  <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                    <Text variant="caption" tone="onPrimary">
                      Principale
                    </Text>
                  </View>
                ) : null}
              </>
            ) : (
              <Ionicons name="add" size={32} color={colors.primary} />
            )}
          </Pressable>
        ))}
        {busy ? (
          <View style={[styles.busy, { backgroundColor: colors.overlay }]}>
            <ActivityIndicator color="#FFFFFF" size="large" />
          </View>
        ) : null}
      </View>
      <Text variant="caption" tone="muted" style={styles.hint}>
        {photos.length}/{MAX_PHOTOS} photos · la première est affichée en premier.
      </Text>
      <ActionMenu
        visible={!!selected}
        title="Cette photo"
        onClose={() => setSelected(null)}
        actions={
          selected
            ? [
                ...(photos[0]?.id !== selected.id
                  ? [{ label: "Définir comme photo principale", onPress: () => void makeMain(selected) }]
                  : []),
                { label: "Remplacer", onPress: () => void replace(selected) },
                {
                  label: "Supprimer",
                  destructive: true,
                  onPress: () => void run(() => datingApi.deletePhoto(selected.id), "Photo supprimée")
                }
              ]
            : []
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: {
    width: "31.5%",
    aspectRatio: 4 / 5,
    borderRadius: radii.md,
    borderWidth: 1.5,
    borderStyle: "dashed",
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  badge: { position: "absolute", left: 6, bottom: 6, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 },
  busy: { ...StyleSheet.absoluteFillObject, alignItems: "center", justifyContent: "center", borderRadius: radii.md },
  hint: { marginTop: spacing.sm }
});
