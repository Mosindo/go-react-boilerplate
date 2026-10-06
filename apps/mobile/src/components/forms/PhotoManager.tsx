import React, { useState } from "react";
import { Alert, Pressable, StyleSheet, View } from "react-native";
import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import { Ionicons } from "@expo/vector-icons";
import { useMutation } from "@tanstack/react-query";
import { deletePhoto, reorderPhotos, uploadPhoto } from "../../api/dating";
import type { Photo } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { describeError } from "../../lib/errors";
import { showToast } from "../../shared/feedback";
import { Badge, Notice, Text, radii, spacing, useTheme } from "../../shared/ui";
import { PhotoImage } from "../PhotoImage";

export const MAX_PHOTOS = 6;
const UPLOAD_WIDTH = 1440;

type PhotoManagerProps = {
  photos: Photo[];
};

async function pickAndPrepare(): Promise<{ uri: string; name: string; type: string } | null> {
  const picked = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 1,
    exif: false
  });
  if (picked.canceled || !picked.assets[0]) {
    return null;
  }
  // Downscale on the device: saves mobile data; the server re-encodes anyway.
  const prepared = await ImageManipulator.manipulateAsync(picked.assets[0].uri, [{ resize: { width: UPLOAD_WIDTH } }], {
    compress: 0.85,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return { uri: prepared.uri, name: "photo.jpg", type: "image/jpeg" };
}

/** Grid of up to six photos: add, replace, delete and choose the primary one (first). */
export function PhotoManager({ photos }: PhotoManagerProps) {
  const { colors } = useTheme();
  const { refreshSelf } = useAuth();
  const [error, setError] = useState<string | null>(null);

  const run = async (action: () => Promise<unknown>) => {
    setError(null);
    try {
      await action();
      await refreshSelf();
    } catch (err) {
      setError(describeError(err, "Opération impossible pour le moment."));
    }
  };

  const upload = useMutation({
    mutationFn: async (replaceId: string | undefined) => {
      const file = await pickAndPrepare();
      if (!file) {
        return;
      }
      await uploadPhoto(file, replaceId);
    }
  });

  const addOrReplace = (replaceId?: string) =>
    run(async () => {
      await upload.mutateAsync(replaceId);
    });

  const makePrimary = (photo: Photo) =>
    run(async () => {
      const ids = [photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)];
      await reorderPhotos(ids);
      showToast("Photo principale mise à jour", "success");
    });

  const remove = (photo: Photo) =>
    Alert.alert("Supprimer cette photo ?", photos.length === 1 ? "Vous devez garder au moins une photo pour être visible." : undefined, [
      { text: "Annuler", style: "cancel" },
      { text: "Supprimer", style: "destructive", onPress: () => void run(() => deletePhoto(photo.id)) }
    ]);

  const openMenu = (photo: Photo) => {
    const buttons = [
      ...(photo.position > 0 ? [{ text: "Définir comme photo principale", onPress: () => void makePrimary(photo) }] : []),
      { text: "Remplacer", onPress: () => void addOrReplace(photo.id) },
      { text: "Supprimer", style: "destructive" as const, onPress: () => remove(photo) },
      { text: "Annuler", style: "cancel" as const }
    ];
    Alert.alert("Photo", undefined, buttons);
  };

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i]);
  const busy = upload.isPending;

  return (
    <View style={styles.root}>
      <View style={styles.grid}>
        {slots.map((photo, index) => (
          <View key={photo?.id ?? `empty-${index}`} style={styles.cell}>
            {photo ? (
              <Pressable
                accessibilityLabel={`Photo ${index + 1}${index === 0 ? ", principale" : ""}. Options`}
                accessibilityRole="button"
                onPress={() => openMenu(photo)}
                style={styles.fill}
              >
                <PhotoImage path={photo.url} style={[styles.fill, { borderRadius: radii.md }]} />
                {index === 0 ? <Badge label="Principale" style={styles.badge} tone="primary" /> : null}
              </Pressable>
            ) : (
              <Pressable
                accessibilityLabel="Ajouter une photo"
                accessibilityRole="button"
                disabled={busy || index !== photos.length}
                onPress={() => void addOrReplace()}
                style={[styles.fill, styles.empty, { borderColor: colors.borderStrong, opacity: index === photos.length ? 1 : 0.4 }]}
                testID={index === photos.length ? "photo-add" : undefined}
              >
                <Ionicons color={colors.primary} name={busy && index === photos.length ? "hourglass-outline" : "add"} size={28} />
              </Pressable>
            )}
          </View>
        ))}
      </View>
      <Text tone="muted" variant="caption">
        {photos.length}/{MAX_PHOTOS} photos · La première est votre photo principale. Touchez une photo pour la modifier.
      </Text>
      {error ? <Notice message={error} tone="danger" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 4 / 5 },
  fill: { flex: 1 },
  empty: { borderWidth: 2, borderStyle: "dashed", borderRadius: radii.md, alignItems: "center", justifyContent: "center" },
  badge: { position: "absolute", left: 6, bottom: 6 }
});
