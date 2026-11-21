import React, { useState } from "react";
import { ActivityIndicator, Platform, Pressable, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { errorMessage } from "../../api/client";
import { photosApi } from "../../api/endpoints";
import type { Photo } from "../../api/types";
import { makeStyles, useTheme } from "../../theme";
import { ActionSheet, confirm, Icon, RemoteImage, showToast, Text, type SheetAction } from "../../ui";
import { useSetMyPhotos } from "./hooks";

const SLOTS = 6;

type PickedImage = { uri: string; mimeType?: string | null };

async function pickImage(source: "library" | "camera"): Promise<PickedImage | null> {
  const options: ImagePicker.ImagePickerOptions = {
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    // Client-side compression; the server also resizes and strips metadata.
    quality: 0.8,
    exif: false
  };
  if (source === "camera") {
    const permission = await ImagePicker.requestCameraPermissionsAsync();
    if (!permission.granted) {
      showToast("Autorisez l'accès à l'appareil photo dans les réglages.", "error");
      return null;
    }
  }
  const result = source === "camera" ? await ImagePicker.launchCameraAsync(options) : await ImagePicker.launchImageLibraryAsync(options);
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  if (asset.fileSize && asset.fileSize > 10 * 1024 * 1024) {
    showToast("Image trop lourde (10 Mo maximum).", "error");
    return null;
  }
  return { uri: asset.uri, mimeType: asset.mimeType };
}

type Props = { photos: Photo[]; minimum?: number };

export function PhotoGrid({ photos, minimum = 1 }: Props) {
  const styles = useStyles();
  const theme = useTheme();
  const setPhotos = useSetMyPhotos();
  const [busySlot, setBusySlot] = useState<number | null>(null);
  const [sheet, setSheet] = useState<{ title: string; actions: SheetAction[] } | null>(null);

  const run = async (slot: number, task: () => Promise<{ photos: Photo[] }>) => {
    setBusySlot(slot);
    try {
      const res = await task();
      setPhotos(res.photos);
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setBusySlot(null);
    }
  };

  const choose = (slot: number, onPicked: (image: PickedImage) => void) => {
    const fromLibrary = async () => {
      const image = await pickImage("library");
      if (image) onPicked(image);
    };
    if (Platform.OS === "web") {
      void fromLibrary();
      return;
    }
    setSheet({
      title: "Ajouter une photo",
      actions: [
        { label: "Choisir dans la galerie", icon: "images-outline", onPress: () => void fromLibrary() },
        {
          label: "Prendre une photo",
          icon: "camera-outline",
          onPress: async () => {
            const image = await pickImage("camera");
            if (image) onPicked(image);
          }
        }
      ]
    });
    void slot;
  };

  const add = (slot: number) => choose(slot, (image) => void run(slot, () => photosApi.upload(image.uri, image.mimeType)));

  const manage = (photo: Photo) => {
    const actions: SheetAction[] = [];
    if (photo.position > 0) {
      actions.push({
        label: "Définir comme photo principale",
        icon: "star-outline",
        testID: "photo-make-main",
        onPress: () => {
          const ordered = [photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)];
          void run(photo.position, () => photosApi.reorder(ordered));
        }
      });
    }
    actions.push({
      label: "Remplacer",
      icon: "swap-horizontal-outline",
      onPress: () =>
        choose(photo.position, (image) => void run(photo.position, () => photosApi.replace(photo.id, image.uri, image.mimeType)))
    });
    actions.push({
      label: "Supprimer",
      icon: "trash-outline",
      destructive: true,
      testID: "photo-delete",
      onPress: async () => {
        if (photos.length <= minimum) {
          showToast("Gardez au moins une photo pour rester visible.", "error");
          return;
        }
        if (await confirm({ title: "Supprimer cette photo ?", confirmLabel: "Supprimer", destructive: true })) {
          void run(photo.position, () => photosApi.remove(photo.id));
        }
      }
    });
    setSheet({ title: "Photo", actions });
  };

  return (
    <View style={styles.grid}>
      {Array.from({ length: SLOTS }).map((_, slot) => {
        const photo = photos[slot];
        const busy = busySlot === slot || (busySlot !== null && !photo && slot === photos.length);
        return (
          <Pressable
            key={photo?.id ?? `empty-${slot}`}
            testID={photo ? `photo-slot-${slot}` : `photo-add-${slot}`}
            accessibilityRole="button"
            accessibilityLabel={photo ? `Photo ${slot + 1}${slot === 0 ? ", principale" : ""}. Modifier` : "Ajouter une photo"}
            disabled={busySlot !== null || (!photo && slot !== photos.length)}
            onPress={() => (photo ? manage(photo) : add(slot))}
            style={({ pressed }) => [styles.slot, slot === 0 ? styles.mainSlot : null, pressed ? styles.pressed : null]}
          >
            {photo ? <RemoteImage uri={photo.thumbUrl} style={styles.image} /> : null}
            {busy ? (
              <View style={styles.overlay}>
                <ActivityIndicator color={theme.colors.primary} />
              </View>
            ) : !photo ? (
              <View style={[styles.addBadge, slot !== photos.length ? styles.addDisabled : null]}>
                <Icon name="add" size={22} color="textOnPrimary" />
              </View>
            ) : null}
            {photo && slot === 0 ? (
              <View style={styles.mainTag}>
                <Text variant="micro" tone="onImage">
                  PRINCIPALE
                </Text>
              </View>
            ) : null}
          </Pressable>
        );
      })}
      <ActionSheet visible={sheet !== null} title={sheet?.title} actions={sheet?.actions ?? []} onClose={() => setSheet(null)} />
    </View>
  );
}

const useStyles = makeStyles((t) => ({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: t.spacing.sm, justifyContent: "space-between" },
  slot: {
    width: "31.5%",
    aspectRatio: 4 / 5,
    borderRadius: t.radii.md,
    backgroundColor: t.colors.surfaceAlt,
    borderWidth: 1,
    borderStyle: "dashed",
    borderColor: t.colors.borderStrong,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  mainSlot: { borderStyle: "solid", borderColor: t.colors.primary },
  pressed: { opacity: 0.8 },
  image: { position: "absolute", top: 0, left: 0, right: 0, bottom: 0 },
  overlay: {
    ...{ position: "absolute", top: 0, left: 0, right: 0, bottom: 0 },
    backgroundColor: t.colors.overlay,
    alignItems: "center",
    justifyContent: "center"
  },
  addBadge: { width: 34, height: 34, borderRadius: 17, backgroundColor: t.colors.primary, alignItems: "center", justifyContent: "center" },
  addDisabled: { opacity: 0.35 },
  mainTag: {
    position: "absolute",
    left: 6,
    bottom: 6,
    backgroundColor: t.colors.scrim,
    borderRadius: t.radii.sm,
    paddingHorizontal: 6,
    paddingVertical: 2
  }
}));
