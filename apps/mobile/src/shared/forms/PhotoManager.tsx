import React, { useMemo, useState } from "react";
import { ActivityIndicator, Image, Pressable, View, useWindowDimensions } from "react-native";
import { resolvePhotoUrl } from "../../api/client";
import type { Photo } from "../../api/models";
import { useDeletePhoto, useReorderPhotos, useUploadPhoto } from "../../hooks/useProfileData";
import { usePhotoPicker } from "../../hooks/usePhotoPicker";
import { messageFromError } from "../../lib/errors";
import { canAddPhoto, makeMain, MAX_PHOTOS, movePhoto, photoIds, sortPhotos } from "../../lib/photos";
import { showToast } from "../feedback/toast";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { ConfirmSheet } from "../ui/ConfirmSheet";
import { Notice } from "../ui/Notice";
import { Sheet } from "../ui/Sheet";
import { Text } from "../ui/Text";
import { useTheme, type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type Props = {
  photos: Photo[];
  /** Photos can never drop below this number (a complete profile needs one). */
  minPhotos?: number;
};

const COLUMNS = 3;
const GAP = 10;
const MAX_CONTENT_WIDTH = 560;
const SIDE_PADDING = 32;

const makeStyles = (t: Theme) => ({
  grid: { flexDirection: "row" as const, flexWrap: "wrap" as const, gap: GAP },
  tile: {
    borderRadius: t.radii.md,
    overflow: "hidden" as const,
    backgroundColor: t.colors.surfaceMuted,
    borderWidth: 1,
    borderColor: t.colors.border,
    alignItems: "center" as const,
    justifyContent: "center" as const
  },
  image: { width: "100%" as const, height: "100%" as const },
  mainBadge: { position: "absolute" as const, left: 6, bottom: 6 },
  empty: { borderStyle: "dashed" as const, borderColor: t.colors.borderStrong },
  uploadingScrim: {
    ...({ position: "absolute", top: 0, left: 0, right: 0, bottom: 0 } as const),
    backgroundColor: t.colors.photoScrim,
    alignItems: "center" as const,
    justifyContent: "center" as const,
    gap: t.spacing.xs
  },
  actions: { gap: t.spacing.sm },
  stack: { gap: t.spacing.md }
});

export function PhotoManager({ minPhotos = 1, photos }: Props) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const { width } = useWindowDimensions();
  const pick = usePhotoPicker();
  const upload = useUploadPhoto();
  const remove = useDeletePhoto();
  const reorder = useReorderPhotos();

  const [error, setError] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [confirmRemove, setConfirmRemove] = useState(false);

  const sorted = useMemo(() => sortPhotos(photos), [photos]);
  const ids = useMemo(() => photoIds(sorted), [sorted]);
  const tileSize = Math.floor(
    (Math.min(width - SIDE_PADDING, MAX_CONTENT_WIDTH) - GAP * (COLUMNS - 1)) / COLUMNS
  );
  const selectedIndex = selectedId ? ids.indexOf(selectedId) : -1;
  const busy = upload.isPending || remove.isPending || reorder.isPending;

  async function handleAdd() {
    setError(null);
    try {
      const picked = await pick();
      if (!picked) {
        return;
      }
      if (picked.tooLarge) {
        setError("That photo is larger than 5 MB. Pick a smaller one or crop it.");
        return;
      }
      await upload.mutateAsync(picked.file);
      showToast("Photo added");
    } catch (err) {
      setError(messageFromError(err, "We could not upload that photo. Please try again."));
    }
  }

  async function applyOrder(next: string[], doneMessage: string) {
    setError(null);
    try {
      await reorder.mutateAsync(next);
      showToast(doneMessage);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setSelectedId(null);
    }
  }

  async function handleRemove() {
    if (!selectedId) {
      return;
    }
    setError(null);
    try {
      await remove.mutateAsync(selectedId);
      showToast("Photo removed");
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setConfirmRemove(false);
      setSelectedId(null);
    }
  }

  const slotCount = Math.max(MAX_PHOTOS, sorted.length);

  return (
    <View style={styles.stack}>
      <View style={styles.grid}>
        {Array.from({ length: slotCount }, (_, index) => {
          const photo = sorted[index];
          const dims = { width: tileSize, height: Math.round(tileSize * 1.25) };
          if (photo) {
            return (
              <Pressable
                accessibilityHint="Opens options to reorder or remove this photo"
                accessibilityLabel={`Photo ${index + 1} of ${sorted.length}${index === 0 ? ", main photo" : ""}`}
                accessibilityRole="button"
                key={photo.id}
                onPress={() => setSelectedId(photo.id)}
                style={[styles.tile, dims]}
                testID={`photo-tile-${index}`}
              >
                <Image
                  accessibilityIgnoresInvertColors
                  source={{ uri: resolvePhotoUrl(photo.url) }}
                  style={styles.image}
                />
                {index === 0 ? <Badge label="Main" size="sm" style={styles.mainBadge} variant="primary" /> : null}
              </Pressable>
            );
          }
          const isNextSlot = index === sorted.length;
          if (isNextSlot && upload.isPending) {
            return (
              <View
                accessibilityLabel="Uploading photo"
                accessibilityRole="progressbar"
                key={`uploading-${index}`}
                style={[styles.tile, dims]}
              >
                <View style={styles.uploadingScrim}>
                  <ActivityIndicator color={theme.colors.primaryForeground} />
                  <Text style={{ color: theme.colors.primaryForeground }} variant="caption" weight="bold">
                    Uploading...
                  </Text>
                </View>
              </View>
            );
          }
          return (
            <Pressable
              accessibilityLabel="Add a photo"
              accessibilityRole="button"
              accessibilityState={{ disabled: busy || !canAddPhoto(sorted.length) }}
              disabled={busy || !isNextSlot}
              key={`empty-${index}`}
              onPress={() => void handleAdd()}
              style={[styles.tile, styles.empty, dims]}
              testID={isNextSlot ? "photo-add-button" : `photo-empty-${index}`}
            >
              <Text tone={isNextSlot ? "primary" : "subtle"} variant="title" weight="bold">
                +
              </Text>
            </Pressable>
          );
        })}
      </View>

      <Text tone="muted" variant="caption">
        {sorted.length} of {MAX_PHOTOS} photos. Tap a photo to make it your main photo, move it, or remove it. JPEG,
        PNG or WebP up to 5 MB.
      </Text>

      {error ? <Notice description={error} title="Photo problem" tone="danger" /> : null}

      <Sheet onClose={() => setSelectedId(null)} title="Photo options" visible={selectedId !== null && !confirmRemove}>
        <View style={styles.actions}>
          <Button
            disabled={selectedIndex <= 0 || busy}
            label="Make main photo"
            onPress={() => selectedId && void applyOrder(makeMain(ids, selectedId), "Main photo updated")}
            testID="photo-make-main"
          />
          <Button
            disabled={selectedIndex <= 0 || busy}
            label="Move earlier"
            onPress={() => void applyOrder(movePhoto(ids, selectedIndex, selectedIndex - 1), "Photo moved")}
            variant="outline"
          />
          <Button
            disabled={selectedIndex < 0 || selectedIndex >= ids.length - 1 || busy}
            label="Move later"
            onPress={() => void applyOrder(movePhoto(ids, selectedIndex, selectedIndex + 1), "Photo moved")}
            variant="outline"
          />
          <Button
            disabled={sorted.length <= minPhotos || busy}
            label="Remove photo"
            onPress={() => setConfirmRemove(true)}
            testID="photo-remove"
            variant="destructive"
          />
          {sorted.length <= minPhotos ? (
            <Text tone="muted" variant="caption">
              Keep at least one photo. Add another one before removing this one.
            </Text>
          ) : null}
        </View>
      </Sheet>

      <ConfirmSheet
        confirmLabel="Remove"
        destructive
        loading={remove.isPending}
        message="This photo will be deleted from Alba. You can add it again later."
        onCancel={() => setConfirmRemove(false)}
        onConfirm={() => void handleRemove()}
        title="Remove this photo?"
        visible={confirmRemove}
      />
    </View>
  );
}
