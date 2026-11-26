import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { errorMessage } from "../../../api/client";
import { deletePhoto, reorderPhotos, replacePhoto, uploadPhoto, type PickedImage } from "../../../api/photos";
import type { Photo } from "../../../api/types";
import { radii, spacing, useTheme } from "../../../theme";
import { ActionSheet, ConfirmDialog, Icon, RemoteImage, Text, showToast } from "../../../ui";

export const MAX_PHOTOS = 6;

async function pickImage(): Promise<PickedImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    showToast("Autorisez l'accès à vos photos dans les réglages pour en ajouter.", "error");
    return null;
  }
  const result = await ImagePicker.launchImageLibraryAsync({ mediaTypes: ["images"], allowsEditing: true, aspect: [4, 5], quality: 1 });
  if (result.canceled || !result.assets[0]) return null;
  const asset = result.assets[0];
  return { uri: asset.uri, width: asset.width, height: asset.height };
}

type Props = { photos: Photo[]; onChange: (photos: Photo[]) => void };

/** Upload, reorder (main photo first), replace and delete profile photos. */
export function PhotoGrid({ photos, onChange }: Props) {
  const { colors } = useTheme();
  const [busySlot, setBusySlot] = useState<number | null>(null);
  const [selected, setSelected] = useState<Photo | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<Photo | null>(null);

  const run = async (slot: number, task: () => Promise<Photo[]>, success?: string) => {
    setBusySlot(slot);
    try {
      onChange(await task());
      if (success) showToast(success, "success");
    } catch (e) {
      showToast(errorMessage(e, "L'opération a échoué."), "error");
    } finally {
      setBusySlot(null);
    }
  };

  const add = async () => {
    const image = await pickImage();
    if (image) await run(photos.length, () => uploadPhoto(image));
  };

  const makeMain = (photo: Photo) =>
    run(0, () => reorderPhotos([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]), "Photo principale mise à jour.");

  const replace = async (photo: Photo) => {
    const image = await pickImage();
    if (image) await run(photo.position, () => replacePhoto(photo.id, image), "Photo remplacée.");
  };

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);
  return (
    <View style={styles.grid} testID="photo-grid">
      {slots.map((photo, index) => {
        const busy = busySlot === index;
        if (!photo) {
          const isNext = index === photos.length;
          return (
            <Pressable
              accessibilityLabel="Ajouter une photo"
              accessibilityRole="button"
              disabled={!isNext || busySlot !== null}
              key={`empty-${index}`}
              onPress={add}
              style={[
                styles.slot,
                styles.empty,
                { borderColor: colors.border, backgroundColor: colors.surfaceMuted, opacity: isNext ? 1 : 0.5 }
              ]}
              testID={isNext ? "photo-add" : undefined}
            >
              {busy ? <ActivityIndicator color={colors.primary} /> : <Icon color={colors.primary} name="add-circle" size={30} />}
            </Pressable>
          );
        }
        return (
          <Pressable
            accessibilityLabel={index === 0 ? "Photo principale, options" : `Photo ${index + 1}, options`}
            accessibilityRole="button"
            disabled={busySlot !== null}
            key={photo.id}
            onPress={() => setSelected(photo)}
            style={styles.slot}
            testID={`photo-slot-${index}`}
          >
            <RemoteImage style={StyleSheet.absoluteFill} url={photo.url} />
            {index === 0 ? (
              <View style={[styles.mainBadge, { backgroundColor: colors.primary }]}>
                <Text style={{ color: colors.onPrimary }} variant="overline">
                  Principale
                </Text>
              </View>
            ) : null}
            {busy ? (
              <View style={[StyleSheet.absoluteFill, styles.busy, { backgroundColor: colors.overlay }]}>
                <ActivityIndicator color="#FFFFFF" />
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
                  ? [{ label: "Définir comme principale", icon: "star-outline" as const, onPress: () => void makeMain(selected) }]
                  : []),
                { label: "Remplacer", icon: "swap-horizontal-outline" as const, onPress: () => void replace(selected) },
                {
                  label: "Supprimer",
                  icon: "trash-outline" as const,
                  destructive: true,
                  onPress: () => setConfirmDelete(selected),
                  testID: "photo-delete"
                }
              ]
            : []
        }
        onClose={() => setSelected(null)}
        title="Photo"
        visible={!!selected}
      />
      <ConfirmDialog
        confirmLabel="Supprimer"
        destructive
        message={
          photos.length === 1
            ? "Sans photo, votre profil ne sera plus visible dans la découverte."
            : "Cette photo sera définitivement supprimée."
        }
        onCancel={() => setConfirmDelete(null)}
        onConfirm={() => {
          const target = confirmDelete;
          setConfirmDelete(null);
          if (target) void run(target.position, () => deletePhoto(target.id), "Photo supprimée.");
        }}
        title="Supprimer la photo ?"
        visible={!!confirmDelete}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: { width: "31%", aspectRatio: 4 / 5, borderRadius: radii.md, overflow: "hidden" },
  empty: { borderWidth: 1.5, borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  mainBadge: { position: "absolute", left: 6, bottom: 6, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 },
  busy: { alignItems: "center", justifyContent: "center" }
});
