import React from "react";
import { Alert, Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { keys } from "../../api/keys";
import { deletePhoto, reorderPhotos, uploadPhoto } from "../../api/profile";
import type { Photo } from "../../api/types";
import { errorMessage } from "../../lib/format";
import { MAX_PHOTOS } from "../../lib/validation";
import { MIN_TOUCH, radii, spacing, useTheme } from "../../theme";
import { AuthImage } from "../AuthImage";
import { Button } from "../Button";
import { Text } from "../Text";
import { useToast } from "../Toast";

export function movePhoto(photos: Photo[], from: number, to: number): string[] {
  const ids = photos.map((p) => p.id);
  if (to < 0 || to >= ids.length || from === to) return ids;
  const [moved] = ids.splice(from, 1);
  if (moved) ids.splice(to, 0, moved);
  return ids;
}

export function PhotosManager({ photos }: { photos: Photo[] }) {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const sorted = [...photos].sort((a, b) => a.position - b.position);

  const refresh = async (): Promise<void> => {
    await queryClient.invalidateQueries({ queryKey: keys.profile });
    await queryClient.invalidateQueries({ queryKey: keys.me });
  };

  const upload = useMutation({
    mutationFn: async () => {
      const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
      if (!permission.granted) throw new Error("Allow photo access in Settings to add pictures.");
      const result = await ImagePicker.launchImageLibraryAsync({
        mediaTypes: ["images"],
        quality: 0.85,
        allowsEditing: true,
        aspect: [4, 5]
      });
      if (result.canceled || !result.assets[0]) return false;
      const asset = result.assets[0];
      await uploadPhoto({
        uri: asset.uri,
        name: asset.fileName ?? "photo.jpg",
        type: asset.mimeType ?? "image/jpeg"
      });
      return true;
    },
    onSuccess: async (added) => {
      if (added) {
        await refresh();
        toast.show("Photo added", "success");
      }
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const reorder = useMutation({
    mutationFn: (ids: string[]) => reorderPhotos(ids),
    onSuccess: refresh,
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const remove = useMutation({
    mutationFn: (id: string) => deletePhoto(id),
    onSuccess: async () => {
      await refresh();
      toast.show("Photo removed");
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const confirmRemove = (photo: Photo): void => {
    Alert.alert("Remove this photo?", undefined, [
      { text: "Cancel", style: "cancel" },
      { text: "Remove", style: "destructive", onPress: () => remove.mutate(photo.id) }
    ]);
  };

  const busy = upload.isPending || reorder.isPending || remove.isPending;
  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => sorted[i] ?? null);

  return (
    <View style={{ gap: spacing.md }}>
      <View style={styles.grid}>
        {slots.map((photo, index) => (
          <View key={photo?.id ?? `empty-${index}`} style={styles.cell}>
            {photo ? (
              <View style={[styles.photo, { borderColor: index === 0 ? colors.accent : colors.border }]}>
                <AuthImage url={photo.url} style={StyleSheet.absoluteFill} label={`Photo ${index + 1}`} />
                {index === 0 ? (
                  <View style={[styles.mainTag, { backgroundColor: colors.accent }]}>
                    <Text variant="caption" style={{ color: "#fff", fontWeight: "700" }}>
                      Main
                    </Text>
                  </View>
                ) : null}
                <View style={styles.actions}>
                  {index > 0 ? (
                    <MiniButton label="Make main photo" glyph="★" onPress={() => reorder.mutate(movePhoto(sorted, index, 0))} disabled={busy} />
                  ) : null}
                  {index < sorted.length - 1 ? (
                    <MiniButton label="Move later" glyph="→" onPress={() => reorder.mutate(movePhoto(sorted, index, index + 1))} disabled={busy} />
                  ) : null}
                  <MiniButton label="Remove photo" glyph="✕" onPress={() => confirmRemove(photo)} disabled={busy} testID={`photo-remove-${index}`} />
                </View>
              </View>
            ) : (
              <Pressable
                accessibilityRole="button"
                accessibilityLabel="Add a photo"
                disabled={busy}
                onPress={() => upload.mutate()}
                testID={index === sorted.length ? "photo-add" : undefined}
                style={[styles.photo, styles.empty, { borderColor: colors.border, backgroundColor: colors.surfaceMuted }]}
              >
                <Text variant="heading" tone="muted">
                  +
                </Text>
              </Pressable>
            )}
          </View>
        ))}
      </View>
      <Text variant="caption" tone="muted">
        Add up to {MAX_PHOTOS} photos. The first one is your main photo. At least one is required.
      </Text>
      {sorted.length < MAX_PHOTOS ? (
        <Button title="Add a photo" variant="secondary" onPress={() => upload.mutate()} loading={upload.isPending} testID="photo-add-button" />
      ) : null}
    </View>
  );
}

function MiniButton({ label, glyph, onPress, disabled, testID }: { label: string; glyph: string; onPress: () => void; disabled?: boolean; testID?: string }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      hitSlop={6}
      style={styles.mini}
    >
      <Text style={{ color: "#fff", fontWeight: "700" }}>{glyph}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 4 / 5 },
  photo: { flex: 1, borderRadius: radii.md, overflow: "hidden", borderWidth: 2 },
  empty: { alignItems: "center", justifyContent: "center", borderStyle: "dashed", minHeight: MIN_TOUCH },
  mainTag: { position: "absolute", top: 6, left: 6, paddingHorizontal: 8, paddingVertical: 2, borderRadius: radii.pill },
  actions: { position: "absolute", bottom: 4, left: 4, right: 4, flexDirection: "row", justifyContent: "flex-end", gap: 4 },
  mini: { width: 30, height: 30, borderRadius: 15, backgroundColor: "rgba(0,0,0,0.6)", alignItems: "center", justifyContent: "center" }
});
