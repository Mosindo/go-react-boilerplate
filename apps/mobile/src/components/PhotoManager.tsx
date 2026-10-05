import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import * as ImagePicker from "expo-image-picker";
import { useState } from "react";
import { Alert, Platform, Pressable, StyleSheet, View } from "react-native";
import { deletePhoto, reorderPhotos, replacePhoto, uploadPhoto } from "../api/platform";
import type { MyProfile, Photo } from "../api/types";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../realtime/cache";
import { showError } from "../shared/feedback";
import { IconButton, Loader, RemoteImage, Text } from "../shared/ui";
import { radii, spacing, useTheme } from "../theme";

const MAX_PHOTOS = 6;

async function pickImage(): Promise<ImagePicker.ImagePickerAsset | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    quality: 0.85, // compression happens on the device first; the server re-encodes anyway
    allowsMultipleSelection: false,
    exif: false
  });
  return result.canceled ? null : (result.assets[0] ?? null);
}

function confirmDelete(onConfirm: () => void) {
  if (Platform.OS === "web") {
    if (globalThis.confirm?.("Supprimer cette photo ?")) onConfirm();
    return;
  }
  Alert.alert("Supprimer cette photo ?", undefined, [
    { text: "Annuler", style: "cancel" },
    { text: "Supprimer", style: "destructive", onPress: onConfirm }
  ]);
}

/**
 * Add, replace, delete and reorder up to 6 photos. The first photo is the main one.
 * Reordering uses explicit "move" buttons (accessible, no drag gesture needed).
 */
export function PhotoManager({ photos }: { photos: Photo[] }) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState<string | null>(null);

  const setPhotos = (next: Photo[]) =>
    queryClient.setQueryData<MyProfile>(queryKeys.me, (data) =>
      data?.profile
        ? {
            ...data,
            profile: { ...data.profile, photos: next },
            completion: {
              ...data.completion,
              photos: next.length > 0,
              complete: data.completion.profile && data.completion.preferences && data.completion.location && next.length > 0
            }
          }
        : data
    );

  const run = async (key: string, action: () => Promise<void>) => {
    setBusy(key);
    try {
      await action();
    } catch (error) {
      showError(errorMessage(error));
    } finally {
      setBusy(null);
      void queryClient.invalidateQueries({ queryKey: queryKeys.me });
    }
  };

  const add = () =>
    run("add", async () => {
      const asset = await pickImage();
      if (!asset) return;
      const created = await uploadPhoto({ uri: asset.uri, fileName: asset.fileName, mimeType: asset.mimeType });
      setPhotos([...photos, created]);
    });

  const replace = (photo: Photo) =>
    run(photo.id, async () => {
      const asset = await pickImage();
      if (!asset) return;
      const updated = await replacePhoto(photo.id, { uri: asset.uri, fileName: asset.fileName, mimeType: asset.mimeType });
      setPhotos(photos.map((p) => (p.id === photo.id ? updated : p)));
    });

  const remove = (photo: Photo) =>
    confirmDelete(
      () =>
        void run(photo.id, async () => {
          await deletePhoto(photo.id);
          setPhotos(photos.filter((p) => p.id !== photo.id).map((p, i) => ({ ...p, position: i })));
        })
    );

  const move = (index: number, to: number) =>
    run("order", async () => {
      const next = [...photos];
      const [item] = next.splice(index, 1);
      next.splice(to, 0, item);
      setPhotos(next.map((p, i) => ({ ...p, position: i })));
      setPhotos(await reorderPhotos(next.map((p) => p.id)));
    });

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);

  return (
    <View style={styles.grid} testID="photo-manager">
      {slots.map((photo, index) => (
        <View key={photo?.id ?? `empty-${index}`} style={styles.cell}>
          {photo ? (
            <View style={[styles.slot, { borderColor: colors.border }]}>
              <RemoteImage path={photo.url} style={StyleSheet.absoluteFill} accessibilityLabel={`Photo ${index + 1}`} />
              {index === 0 ? (
                <View style={[styles.main, { backgroundColor: colors.primary }]}>
                  <Text variant="label" tone="onPrimary" style={styles.mainText}>
                    Principale
                  </Text>
                </View>
              ) : null}
              {busy === photo.id ? (
                <View style={[StyleSheet.absoluteFill, styles.busy, { backgroundColor: colors.overlay }]}>
                  <Loader />
                </View>
              ) : null}
              <View style={styles.actions}>
                {index > 0 ? (
                  <MiniButton
                    icon="arrow-back"
                    label={index === 1 ? "Mettre en photo principale" : "Déplacer vers la gauche"}
                    onPress={() => move(index, index - 1)}
                  />
                ) : null}
                {index < photos.length - 1 ? (
                  <MiniButton icon="arrow-forward" label="Déplacer vers la droite" onPress={() => move(index, index + 1)} />
                ) : null}
                <MiniButton icon="swap-horizontal" label="Remplacer la photo" onPress={() => void replace(photo)} />
                <MiniButton icon="trash-outline" label="Supprimer la photo" onPress={() => remove(photo)} />
              </View>
            </View>
          ) : (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Ajouter une photo"
              testID={index === photos.length ? "photo-add" : undefined}
              disabled={index !== photos.length || busy !== null}
              onPress={() => void add()}
              style={[
                styles.slot,
                styles.empty,
                { borderColor: colors.border, backgroundColor: colors.surface, opacity: index === photos.length ? 1 : 0.5 }
              ]}
            >
              {busy === "add" && index === photos.length ? <Loader /> : <Ionicons name="add" size={32} color={colors.primary} />}
            </Pressable>
          )}
        </View>
      ))}
    </View>
  );
}

function MiniButton({ icon, label, onPress }: { icon: keyof typeof Ionicons.glyphMap; label: string; onPress: () => void }) {
  return (
    <View style={styles.mini}>
      <IconButton icon={icon} label={label} onPress={onPress} size={16} color="#FFFFFF" style={styles.miniButton} />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", marginHorizontal: -spacing.xs },
  cell: { width: "33.333%", padding: spacing.xs },
  slot: { aspectRatio: 3 / 4, borderRadius: radii.md, overflow: "hidden", borderWidth: 1 },
  empty: { alignItems: "center", justifyContent: "center", borderStyle: "dashed", borderWidth: 1.5 },
  main: { position: "absolute", top: 6, left: 6, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 },
  mainText: { fontSize: 11, lineHeight: 15 },
  busy: { alignItems: "center", justifyContent: "center" },
  actions: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    flexWrap: "wrap",
    justifyContent: "center",
    backgroundColor: "rgba(0,0,0,0.45)",
    paddingVertical: 2
  },
  mini: { width: 30, height: 30, alignItems: "center", justifyContent: "center" },
  miniButton: { width: 30, height: 30 }
});
