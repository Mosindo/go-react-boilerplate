import React, { useState } from "react";
import { Alert, StyleSheet, View } from "react-native";
import type { Photo } from "../api/types";
import { errorMessage } from "../api/client";
import { MAX_PHOTOS, moveItem } from "../domain/photos";
import { useDeletePhoto, useReorderPhotos, useUploadPhoto } from "../hooks/useProfile";
import { pickPhotoForUpload } from "../lib/photoPicker";
import { showToast } from "../shared/feedback";
import { Button, Loader, PhotoImage, Text } from "../shared/ui";
import { radius, spacing, useTheme } from "../theme";

function PhotoTile({
  photo,
  index,
  total,
  busy,
  onMove,
  onDelete
}: {
  photo: Photo;
  index: number;
  total: number;
  busy: boolean;
  onMove: (index: number, delta: number) => void;
  onDelete: (photo: Photo) => void;
}) {
  const theme = useTheme();
  return (
    <View style={styles.tile}>
      <View style={[styles.frame, { borderColor: index === 0 ? theme.primary : theme.border }]}>
        <PhotoImage
          accessibilityLabel={`Photo ${index + 1}${index === 0 ? ", main photo" : ""}`}
          photo={photo}
          style={styles.image}
        />
        {index === 0 ? (
          <View style={[styles.mainTag, { backgroundColor: theme.primary }]}>
            <Text style={{ color: theme.onPrimary }} variant="caption">
              Main
            </Text>
          </View>
        ) : null}
      </View>
      <View style={styles.actions}>
        {index > 0 ? (
          <Button
            a11yLabel={`Make photo ${index + 1} the main photo`}
            disabled={busy}
            label="Make main"
            onPress={() => onMove(index, -index)}
            variant="secondary"
          />
        ) : null}
        {index > 1 ? (
          <Button
            a11yLabel={`Move photo ${index + 1} earlier`}
            disabled={busy}
            label="Move earlier"
            onPress={() => onMove(index, -1)}
            variant="secondary"
          />
        ) : null}
        {index < total - 1 ? (
          <Button
            a11yLabel={`Move photo ${index + 1} later`}
            disabled={busy}
            label="Move later"
            onPress={() => onMove(index, 1)}
            variant="secondary"
          />
        ) : null}
        <Button
          a11yLabel={`Delete photo ${index + 1}`}
          disabled={busy || total <= 1}
          label="Delete"
          onPress={() => onDelete(photo)}
          variant="ghost"
        />
      </View>
    </View>
  );
}

export function PhotoManager({ photos }: { photos: Photo[] }) {
  const upload = useUploadPhoto();
  const remove = useDeletePhoto();
  const reorder = useReorderPhotos();
  const [preparing, setPreparing] = useState(false);
  const ordered = [...photos].sort((a, b) => a.position - b.position);
  const busy = preparing || upload.isPending || remove.isPending || reorder.isPending;

  const add = async () => {
    setPreparing(true);
    try {
      const file = await pickPhotoForUpload();
      if (file) {
        await upload.mutateAsync(file);
        showToast("Photo added.", "success");
      }
    } catch (error) {
      showToast(errorMessage(error, "We could not add that photo."), "error");
    } finally {
      setPreparing(false);
    }
  };

  const move = (index: number, delta: number) => {
    const next = moveItem(ordered, index, delta).map((photo) => photo.id);
    reorder.mutate(next, { onError: (error) => showToast(errorMessage(error, "Could not reorder photos."), "error") });
  };

  const confirmDelete = (photo: Photo) => {
    Alert.alert("Delete this photo?", "This cannot be undone.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Delete",
        style: "destructive",
        onPress: () =>
          remove.mutate(photo.id, {
            onError: (error) => showToast(errorMessage(error, "Could not delete the photo."), "error")
          })
      }
    ]);
  };

  return (
    <View style={styles.wrap}>
      <Text tone="muted">
        Add up to {MAX_PHOTOS} photos. The first one is your main photo. At least one is required.
      </Text>
      {ordered.map((photo, index) => (
        <PhotoTile
          busy={busy}
          index={index}
          key={photo.id}
          onDelete={confirmDelete}
          onMove={move}
          photo={photo}
          total={ordered.length}
        />
      ))}
      {busy ? <Loader /> : null}
      {ordered.length < MAX_PHOTOS ? (
        <Button
          disabled={busy}
          label={ordered.length === 0 ? "Add your first photo" : "Add a photo"}
          onPress={() => void add()}
          variant={ordered.length === 0 ? "primary" : "secondary"}
        />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.lg },
  tile: { gap: spacing.sm },
  frame: { borderRadius: radius.lg, borderWidth: 2, overflow: "hidden", aspectRatio: 4 / 3 },
  image: { width: "100%", height: "100%" },
  mainTag: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: 2,
    borderRadius: radius.pill
  },
  actions: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
