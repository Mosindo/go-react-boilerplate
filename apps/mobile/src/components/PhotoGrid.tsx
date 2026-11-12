import React, { useState } from "react";
import { ActivityIndicator, Alert, Platform, Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { errorMessage } from "../api/client";
import { photosApi, type LocalImage } from "../api/endpoints";
import type { Photo as PhotoT } from "../api/types";
import { radii, spacing, useTheme } from "../theme";
import { Icon, Photo, Text, showToast } from "../ui";

const SLOTS = 6;

async function pickImage(): Promise<LocalImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted && Platform.OS !== "web") {
    showToast("Autorisez l'accès à vos photos dans les réglages.", "error");
    return null;
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.82,
    exif: false
  });
  if (result.canceled || !result.assets[0]) return null;
  const a = result.assets[0];
  if (a.fileSize && a.fileSize > 10 * 1024 * 1024) {
    showToast("Photo trop lourde (10 Mo max).", "error");
    return null;
  }
  return { uri: a.uri, mimeType: a.mimeType, fileName: a.fileName, file: (a as { file?: File }).file };
}

function confirm(title: string, message: string, actionLabel: string): Promise<boolean> {
  if (Platform.OS === "web") {
    return Promise.resolve(globalThis.confirm?.(`${title}\n\n${message}`) ?? true);
  }
  return new Promise((resolve) =>
    Alert.alert(title, message, [
      { text: "Annuler", style: "cancel", onPress: () => resolve(false) },
      { text: actionLabel, style: "destructive", onPress: () => resolve(true) }
    ])
  );
}

/**
 * Six-slot photo manager: add, replace (tap), set as main, delete.
 * The first photo is the main one shown in discovery.
 */
export function PhotoGrid({ photos, onChange }: { photos: PhotoT[]; onChange: (photos: PhotoT[]) => void }) {
  const { colors } = useTheme();
  const [busySlot, setBusySlot] = useState<number | null>(null);

  const run = async (slot: number, action: () => Promise<PhotoT[]>, success?: string) => {
    setBusySlot(slot);
    try {
      onChange(await action());
      if (success) showToast(success, "success");
    } catch (e) {
      showToast(errorMessage(e, "Impossible d'enregistrer la photo."), "error");
    } finally {
      setBusySlot(null);
    }
  };

  const add = async (slot: number) => {
    const image = await pickImage();
    if (image) await run(slot, () => photosApi.upload(image));
  };

  const manage = async (photo: PhotoT, slot: number) => {
    if (Platform.OS === "web") {
      const image = await pickImage();
      if (image) await run(slot, () => photosApi.replace(photo.id, image), "Photo remplacée");
      return;
    }
    Alert.alert("Photo", undefined, [
      ...(slot > 0
        ? [
            {
              text: "Mettre en photo principale",
              onPress: () =>
                void run(slot, () => photosApi.reorder([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]), "Photo principale mise à jour")
            }
          ]
        : []),
      {
        text: "Remplacer",
        onPress: async () => {
          const image = await pickImage();
          if (image) await run(slot, () => photosApi.replace(photo.id, image), "Photo remplacée");
        }
      },
      {
        text: "Supprimer",
        style: "destructive" as const,
        onPress: async () => {
          const last = photos.length === 1;
          const ok = await confirm(
            "Supprimer cette photo ?",
            last ? "Sans photo, votre profil n'apparaîtra plus dans la découverte." : "Cette action est définitive.",
            "Supprimer"
          );
          if (ok) await run(slot, () => photosApi.remove(photo.id), "Photo supprimée");
        }
      },
      { text: "Annuler", style: "cancel" as const }
    ]);
  };

  return (
    <View style={styles.grid}>
      {Array.from({ length: SLOTS }).map((_, slot) => {
        const photo = photos[slot];
        const busy = busySlot === slot;
        return (
          <Pressable
            key={photo?.id ?? `empty-${slot}`}
            accessibilityRole="button"
            accessibilityLabel={photo ? `Photo ${slot + 1}${slot === 0 ? ", principale" : ""}. Modifier` : "Ajouter une photo"}
            onPress={() => (photo ? void manage(photo, slot) : void add(slot))}
            disabled={busySlot !== null || (!photo && slot > photos.length)}
            style={[
              styles.slot,
              { backgroundColor: colors.surfaceMuted, borderColor: colors.border },
              !photo && slot > photos.length ? { opacity: 0.5 } : null
            ]}
            testID={photo ? `photo-slot-${slot}` : `photo-add-${slot}`}
          >
            {photo ? <Photo url={photo.url} style={StyleSheet.absoluteFill} /> : <Icon name="add" size={30} color={colors.primary} />}
            {slot === 0 && photo ? (
              <View style={[styles.mainTag, { backgroundColor: colors.primary }]}>
                <Text variant="caption" style={{ color: colors.onPrimary, fontWeight: "700" }}>
                  Principale
                </Text>
              </View>
            ) : null}
            {busy ? (
              <View style={[StyleSheet.absoluteFill, styles.busy, { backgroundColor: colors.overlay }]}>
                <ActivityIndicator color="#fff" />
              </View>
            ) : null}
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: {
    width: "31%",
    aspectRatio: 4 / 5,
    borderRadius: radii.md,
    borderWidth: 1,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center",
    flexGrow: 1,
    maxWidth: "32%"
  },
  mainTag: { position: "absolute", left: 6, bottom: 6, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 },
  busy: { alignItems: "center", justifyContent: "center" }
});
