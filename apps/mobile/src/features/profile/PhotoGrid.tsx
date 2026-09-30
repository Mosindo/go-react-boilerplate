import React, { useState } from "react";
import { ActivityIndicator, Platform, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import * as ImagePicker from "expo-image-picker";
import { Ionicons } from "@expo/vector-icons";
import { ActionSheet, Text, type SheetAction } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { mediaUrl } from "../../lib/api/config";
import { photosApi } from "../../lib/api/endpoints";
import type { Photo } from "../../lib/api/types";
import { showToast } from "../../lib/toast";
import { usePhotosCacheUpdater } from "./hooks";

const MAX_PHOTOS = 6;

async function pickImage(): Promise<ImagePicker.ImagePickerAsset | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted && Platform.OS !== "web") {
    showToast("Autorisez l'accès à vos photos dans les réglages.", "error");
    return null;
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    // Client-side compression; the server re-encodes and strips metadata too.
    quality: 0.8,
    exif: false
  });
  return result.canceled ? null : result.assets[0] ?? null;
}

export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const { colors, radii } = useTheme();
  const updateCache = usePhotosCacheUpdater();
  const [busySlot, setBusySlot] = useState<number | null>(null);
  const [sheet, setSheet] = useState<{ photo: Photo; index: number } | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<{ photo: Photo; index: number } | null>(null);

  const run = async (slot: number, task: () => Promise<Photo[]>, success?: string) => {
    setBusySlot(slot);
    try {
      updateCache(await task());
      if (success) showToast(success, "success");
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setBusySlot(null);
    }
  };

  const add = async (slot: number) => {
    const asset = await pickImage();
    if (asset) await run(slot, () => photosApi.upload(asset.uri, asset.mimeType), "Photo ajoutée");
  };

  const setMain = (photo: Photo, index: number) => {
    const order = [photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)];
    void run(index, () => photosApi.reorder(order), "Photo principale mise à jour");
  };

  const replace = async (photo: Photo, index: number) => {
    const asset = await pickImage();
    if (asset) await run(index, () => photosApi.replace(photo.id, asset.uri, asset.mimeType), "Photo remplacée");
  };

  const sheetActions: SheetAction[] = sheet
    ? [
        ...(sheet.index > 0 ? [{ label: "Définir comme principale", onPress: () => setMain(sheet.photo, sheet.index), testID: "photo-action-main" }] : []),
        { label: "Remplacer", onPress: () => void replace(sheet.photo, sheet.index), testID: "photo-action-replace" },
        { label: "Supprimer", destructive: true, onPress: () => setConfirmDelete(sheet), testID: "photo-action-delete" }
      ]
    : [];

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);
  return (
    <View style={styles.grid} testID="photo-grid">
      {slots.map((photo, index) => (
        <Pressable
          accessibilityLabel={photo ? `Photo ${index + 1}${index === 0 ? ", principale" : ""}. Modifier` : `Ajouter une photo`}
          accessibilityRole="button"
          disabled={busySlot !== null || (!photo && index !== photos.length)}
          key={photo?.id ?? `empty-${index}`}
          onPress={() => (photo ? setSheet({ photo, index }) : void add(index))}
          style={[
            styles.slot,
            {
              borderRadius: radii.md,
              backgroundColor: colors.surfaceMuted,
              borderColor: photo ? "transparent" : colors.border,
              opacity: !photo && index !== photos.length ? 0.5 : 1
            }
          ]}
          testID={photo ? `photo-slot-${index}` : index === photos.length ? "photo-add" : undefined}
        >
          {photo ? (
            <Image cachePolicy="memory-disk" contentFit="cover" source={{ uri: mediaUrl(photo.url) }} style={StyleSheet.absoluteFill} />
          ) : (
            <Ionicons color={colors.primary} name="add-circle" size={30} />
          )}
          {index === 0 && photo ? (
            <View style={[styles.mainBadge, { backgroundColor: colors.primary }]}>
              <Text style={{ color: colors.onPrimary }} variant="overline">
                PRINCIPALE
              </Text>
            </View>
          ) : null}
          {busySlot === index ? (
            <View style={[StyleSheet.absoluteFill, styles.busy, { backgroundColor: colors.overlay }]}>
              <ActivityIndicator color="#fff" />
            </View>
          ) : null}
        </Pressable>
      ))}
      <ActionSheet actions={sheetActions} onClose={() => setSheet(null)} title="Photo" visible={sheet !== null} />
      <ActionSheet
        actions={
          confirmDelete
            ? [{ label: "Supprimer", destructive: true, onPress: () => void run(confirmDelete.index, () => photosApi.remove(confirmDelete.photo.id)), testID: "photo-confirm-delete" }]
            : []
        }
        message="Elle sera définitivement supprimée."
        onClose={() => setConfirmDelete(null)}
        title="Supprimer cette photo ?"
        visible={confirmDelete !== null}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  slot: {
    width: "31%",
    aspectRatio: 4 / 5,
    borderWidth: 1.5,
    borderStyle: "dashed",
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden"
  },
  mainBadge: { position: "absolute", bottom: 6, left: 6, paddingHorizontal: 6, paddingVertical: 2, borderRadius: 6 },
  busy: { alignItems: "center", justifyContent: "center" }
});
