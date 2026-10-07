import * as ImagePicker from "expo-image-picker";
import React, { useMemo, useState } from "react";
import { Alert, Image, Linking, Pressable, StyleSheet, useWindowDimensions, View } from "react-native";
import {
  deletePhoto,
  replacePhoto,
  reorderPhotos,
  uploadPhoto,
  type UploadableImage
} from "../../api/photos";
import { resolveMediaUrl } from "../../api/client";
import type { Photo } from "../../api/types";
import { strings } from "../../i18n/strings";
import { reportError, showToast } from "../../shared/feedback";
import { Badge, Button, Text, radii, spacing } from "../../shared/ui";
import { useTheme } from "../../theme/useTheme";
import { MAX_PHOTOS } from "../../utils/validation";

export const MAX_PHOTO_BYTES = 5 * 1024 * 1024;
const ACCEPTED_TYPES = ["image/jpeg", "image/jpg", "image/png"];

/** Returns an error message when the picked asset cannot be uploaded, otherwise null. */
export function validatePickedImage(asset: {
  mimeType?: string | null;
  fileSize?: number | null;
}): string | null {
  if (asset.mimeType && !ACCEPTED_TYPES.includes(asset.mimeType.toLowerCase())) {
    return strings.photos.unsupported;
  }
  if (asset.fileSize && asset.fileSize > MAX_PHOTO_BYTES) {
    return strings.photos.tooLarge;
  }
  return null;
}

/** Moves `id` to `toIndex`, returning the new order of ids. */
export function moveId(ids: string[], id: string, toIndex: number): string[] {
  const from = ids.indexOf(id);
  if (from === -1 || toIndex < 0 || toIndex >= ids.length) {
    return ids;
  }
  const next = ids.filter((value) => value !== id);
  next.splice(toIndex, 0, id);
  return next;
}

async function pickImage(): Promise<UploadableImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted && !permission.canAskAgain) {
    Alert.alert(strings.photos.permissionTitle, strings.photos.permissionBody, [
      { text: strings.common.cancel, style: "cancel" },
      { text: strings.common.openSettings, onPress: () => void Linking.openSettings() }
    ]);
    return null;
  }
  if (!permission.granted) {
    return null;
  }

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
  const problem = validatePickedImage(asset);
  if (problem) {
    showToast(problem, "error");
    return null;
  }
  return { uri: asset.uri, mimeType: asset.mimeType, fileName: asset.fileName };
}

type PhotoManagerProps = {
  photos: Photo[];
  onChange: (photos: Photo[]) => void;
};

const COLUMNS = 3;

export function PhotoManager({ onChange, photos }: PhotoManagerProps) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const [busy, setBusy] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const slotWidth = useMemo(() => {
    const available = Math.min(width, 560) - spacing.lg * 2 - spacing.sm * (COLUMNS - 1);
    return Math.floor(available / COLUMNS);
  }, [width]);

  const sorted = useMemo(() => [...photos].sort((a, b) => a.position - b.position), [photos]);
  const selectedIndex = sorted.findIndex((photo) => photo.id === selectedId);
  const selected = selectedIndex >= 0 ? sorted[selectedIndex] : null;

  async function run(action: () => Promise<void>) {
    setBusy(true);
    try {
      await action();
    } catch (error) {
      reportError(error);
    } finally {
      setBusy(false);
    }
  }

  const add = () =>
    run(async () => {
      const image = await pickImage();
      if (!image) {
        return;
      }
      const created = await uploadPhoto(image);
      onChange([...sorted, created]);
      setSelectedId(null);
    });

  const replace = () =>
    run(async () => {
      if (!selected) {
        return;
      }
      const image = await pickImage();
      if (!image) {
        return;
      }
      const updated = await replacePhoto(selected.id, image);
      onChange(sorted.map((photo) => (photo.id === selected.id ? updated : photo)));
    });

  const move = (toIndex: number) =>
    run(async () => {
      if (!selected) {
        return;
      }
      const ids = moveId(
        sorted.map((photo) => photo.id),
        selected.id,
        toIndex
      );
      onChange(await reorderPhotos(ids));
    });

  const remove = () => {
    if (!selected) {
      return;
    }
    if (sorted.length <= 1) {
      showToast(strings.photos.keepOne, "info");
      return;
    }
    Alert.alert(strings.photos.deleteTitle, strings.photos.deleteBody, [
      { text: strings.common.cancel, style: "cancel" },
      {
        text: strings.common.delete,
        style: "destructive",
        onPress: () =>
          void run(async () => {
            await deletePhoto(selected.id);
            onChange(sorted.filter((photo) => photo.id !== selected.id));
            setSelectedId(null);
          })
      }
    ]);
  };

  const slots: (Photo | null)[] = [...sorted];
  while (slots.length < MAX_PHOTOS) {
    slots.push(null);
  }

  return (
    <View style={styles.root}>
      <View style={styles.grid}>
        {slots.map((photo, index) => {
          const boxStyle = { width: slotWidth, height: Math.round(slotWidth * 1.25), borderRadius: radii.md };
          if (!photo) {
            const isNext = index === sorted.length;
            return (
              <Pressable
                accessibilityLabel={strings.photos.add}
                accessibilityRole="button"
                disabled={busy || !isNext}
                key={`empty-${index}`}
                onPress={() => void add()}
                style={[
                  styles.empty,
                  boxStyle,
                  {
                    borderColor: colors.borderStrong,
                    backgroundColor: colors.surfaceMuted,
                    opacity: isNext ? 1 : 0.5
                  }
                ]}
              >
                <Text tone="primary" variant="heading">
                  +
                </Text>
              </Pressable>
            );
          }
          const isSelected = photo.id === selectedId;
          return (
            <Pressable
              accessibilityLabel={strings.photos.photoN(index + 1)}
              accessibilityRole="button"
              key={photo.id}
              onPress={() => setSelectedId(isSelected ? null : photo.id)}
              style={[
                boxStyle,
                styles.photoWrap,
                { borderColor: isSelected ? colors.primary : "transparent" }
              ]}
            >
              <Image
                source={{ uri: resolveMediaUrl(photo.url) }}
                style={[styles.image, { borderRadius: radii.md - 2 }]}
              />
              {index === 0 ? (
                <View style={styles.mainBadge}>
                  <Badge label={strings.photos.main} size="sm" variant="primary" />
                </View>
              ) : null}
            </Pressable>
          );
        })}
      </View>

      <Text tone="muted" variant="caption">
        {strings.photos.hint(sorted.length, MAX_PHOTOS)}
      </Text>

      {selected ? (
        <View style={styles.actions}>
          {selectedIndex > 0 ? (
            <Button
              disabled={busy}
              label={strings.photos.makeMain}
              onPress={() => void move(0)}
              size="sm"
              variant="secondary"
            />
          ) : null}
          {selectedIndex > 0 ? (
            <Button
              disabled={busy}
              label={strings.photos.earlier}
              onPress={() => void move(selectedIndex - 1)}
              size="sm"
              variant="outline"
            />
          ) : null}
          {selectedIndex < sorted.length - 1 ? (
            <Button
              disabled={busy}
              label={strings.photos.later}
              onPress={() => void move(selectedIndex + 1)}
              size="sm"
              variant="outline"
            />
          ) : null}
          <Button
            disabled={busy}
            label={strings.photos.replace}
            onPress={() => void replace()}
            size="sm"
            variant="outline"
          />
          <Button
            disabled={busy}
            label={strings.common.delete}
            onPress={remove}
            size="sm"
            variant="destructive"
          />
        </View>
      ) : sorted.length > 0 ? (
        <Text tone="muted" variant="caption">
          {strings.photos.tapToManage}
        </Text>
      ) : null}

      {sorted.length < MAX_PHOTOS ? (
        <Button
          label={sorted.length === 0 ? strings.photos.addFirst : strings.photos.add}
          loading={busy}
          onPress={() => void add()}
          variant={sorted.length === 0 ? "primary" : "secondary"}
        />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  empty: { borderWidth: 1, borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  photoWrap: { borderWidth: 2, padding: 2 },
  image: { width: "100%", height: "100%" },
  mainBadge: { position: "absolute", left: 6, top: 6 },
  actions: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
