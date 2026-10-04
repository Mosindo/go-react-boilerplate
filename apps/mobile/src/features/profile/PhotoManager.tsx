import * as ImagePicker from "expo-image-picker";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { getErrorMessage } from "../../api/client";
import { MAX_PHOTOS } from "../../api/profile";
import type { Photo } from "../../api/types";
import { useDeletePhoto, useReorderPhotos, useUploadPhoto } from "../../hooks/useProfile";
import { moveId, validatePickedPhoto } from "../../lib/photos";
import { MIN_TOUCH, radii, spacing, useColors } from "../../theme";
import { ConfirmDialog, Icon, ProfilePhoto, Text, toast } from "../../ui";

type Props = { photos: Photo[]; name?: string };

export function PhotoManager({ photos, name }: Props) {
  const colors = useColors();
  const upload = useUploadPhoto();
  const remove = useDeletePhoto();
  const reorder = useReorderPhotos();
  const [pendingDelete, setPendingDelete] = React.useState<Photo | null>(null);
  const busy = upload.isPending || remove.isPending || reorder.isPending;

  async function pick() {
    try {
      const result = await ImagePicker.launchImageLibraryAsync({
        mediaTypes: ["images"],
        allowsEditing: true,
        aspect: [4, 5],
        quality: 0.85
      });
      if (result.canceled || result.assets.length === 0) return;
      const asset = result.assets[0];
      const problem = validatePickedPhoto(asset);
      if (problem) {
        toast.error(problem);
        return;
      }
      await upload.mutateAsync({ uri: asset.uri, mimeType: asset.mimeType, fileName: asset.fileName });
      toast.success("Photo added");
    } catch (error) {
      toast.error(getErrorMessage(error, "We couldn't add that photo."));
    }
  }

  async function move(from: number, to: number) {
    try {
      await reorder.mutateAsync(moveId(photos, from, to));
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  }

  async function confirmDelete() {
    const photo = pendingDelete;
    if (!photo) return;
    try {
      await remove.mutateAsync(photo.id);
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setPendingDelete(null);
    }
  }

  const sorted = [...photos].sort((a, b) => a.position - b.position);
  const slots = Array.from({ length: MAX_PHOTOS }, (_, index) => sorted[index] ?? null);

  return (
    <View style={styles.wrap}>
      <View style={styles.grid}>
        {slots.map((photo, index) =>
          photo ? (
            <View key={photo.id} style={styles.slot}>
              <ProfilePhoto name={name} photo={photo} size="thumb" style={styles.image} />
              {index === 0 ? (
                <View style={[styles.mainTag, { backgroundColor: colors.primary }]}>
                  <Text style={{ color: colors.onPrimary }} variant="caption" weight="bold">
                    Main
                  </Text>
                </View>
              ) : null}
              <Pressable
                accessibilityLabel={`Delete photo ${index + 1}`}
                accessibilityRole="button"
                disabled={busy}
                hitSlop={6}
                onPress={() => setPendingDelete(photo)}
                style={[styles.corner, styles.topRight, { backgroundColor: colors.surface }]}
              >
                <Icon color={colors.danger} name="trash-outline" size={18} />
              </Pressable>
              <View style={styles.moveRow}>
                {index > 0 ? (
                  <Pressable
                    accessibilityLabel={
                      index === 1 ? `Make photo ${index + 1} the main photo` : `Move photo ${index + 1} earlier`
                    }
                    accessibilityRole="button"
                    disabled={busy}
                    hitSlop={6}
                    onPress={() => move(index, index - 1)}
                    style={[styles.corner, { backgroundColor: colors.surface }]}
                  >
                    <Icon name="arrow-back" size={18} />
                  </Pressable>
                ) : (
                  <View />
                )}
                {index < sorted.length - 1 ? (
                  <Pressable
                    accessibilityLabel={`Move photo ${index + 1} later`}
                    accessibilityRole="button"
                    disabled={busy}
                    hitSlop={6}
                    onPress={() => move(index, index + 1)}
                    style={[styles.corner, { backgroundColor: colors.surface }]}
                  >
                    <Icon name="arrow-forward" size={18} />
                  </Pressable>
                ) : null}
              </View>
            </View>
          ) : (
            <Pressable
              key={`empty-${index}`}
              accessibilityLabel={index === sorted.length ? "Add a photo" : "Empty photo slot"}
              accessibilityRole="button"
              accessibilityState={{ disabled: busy || index !== sorted.length }}
              disabled={busy || index !== sorted.length}
              onPress={pick}
              style={[
                styles.slot,
                styles.empty,
                { borderColor: colors.border, backgroundColor: colors.surfaceMuted },
                index !== sorted.length && { opacity: 0.5 }
              ]}
            >
              <Icon
                color={colors.primary}
                name={upload.isPending && index === sorted.length ? "hourglass-outline" : "add"}
                size={28}
              />
            </Pressable>
          )
        )}
      </View>
      <Text tone="muted" variant="small">
        {photos.length} of {MAX_PHOTOS} photos. The first photo is your main one. Use the arrows to reorder.
      </Text>
      <ConfirmDialog
        confirmLabel="Delete photo"
        destructive
        loading={remove.isPending}
        message="This photo will be removed from your profile."
        onCancel={() => setPendingDelete(null)}
        onConfirm={confirmDelete}
        title="Delete this photo?"
        visible={pendingDelete !== null}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.sm },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  slot: { width: "31.5%", aspectRatio: 4 / 5, borderRadius: radii.md, overflow: "hidden" },
  image: { width: "100%", height: "100%" },
  empty: {
    borderWidth: 1.5,
    borderStyle: "dashed",
    alignItems: "center",
    justifyContent: "center",
    minHeight: MIN_TOUCH
  },
  mainTag: {
    position: "absolute",
    left: 6,
    top: 6,
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 2
  },
  corner: {
    width: 32,
    height: 32,
    borderRadius: 16,
    alignItems: "center",
    justifyContent: "center",
    opacity: 0.95
  },
  topRight: { position: "absolute", right: 6, top: 6 },
  moveRow: {
    position: "absolute",
    left: 6,
    right: 6,
    bottom: 6,
    flexDirection: "row",
    justifyContent: "space-between"
  }
});
