import React from "react";
import { Alert, Pressable, StyleSheet, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import type { MyProfile, Photo } from "../api/types";
import { keys } from "../state/keys";
import { AuthImage } from "../ui/AuthImage";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";
import { radii, usePalette } from "../theme";

export const MAX_PHOTOS = 6;

export function buildPhotoForm(asset: {
  uri: string;
  fileName?: string | null;
  mimeType?: string | null;
}): FormData {
  const form = new FormData();
  const type = asset.mimeType === "image/png" ? "image/png" : "image/jpeg";
  const name = asset.fileName ?? (type === "image/png" ? "photo.png" : "photo.jpg");
  // React Native's FormData accepts a { uri, name, type } descriptor in place of a Blob.
  form.append("photo", { uri: asset.uri, name, type } as unknown as Blob);
  return form;
}

/** Add, replace, delete and reorder profile photos. Position 0 is the main photo. */
export function PhotoManager({ photos }: { photos: Photo[] }) {
  const p = usePalette();
  const toast = useToast();
  const queryClient = useQueryClient();

  const onDone = (profile?: MyProfile) => {
    if (profile) queryClient.setQueryData(keys.profile, profile);
    else void queryClient.invalidateQueries({ queryKey: keys.profile });
  };
  const onError = (error: Error) => toast.show(error.message, "error");

  const pick = async (): Promise<FormData | null> => {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      toast.show("Allow photo access to add pictures.", "error");
      return null;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"],
      allowsEditing: true,
      aspect: [4, 5],
      quality: 0.85
    });
    const asset = result.canceled ? undefined : result.assets[0];
    return asset ? buildPhotoForm(asset) : null;
  };

  const add = useMutation({
    mutationFn: async () => {
      const form = await pick();
      if (form) await profileApi.addPhoto(form);
    },
    onSuccess: () => onDone(),
    onError
  });
  const replace = useMutation({
    mutationFn: async (id: string) => {
      const form = await pick();
      if (form) await profileApi.replacePhoto(id, form);
    },
    onSuccess: () => onDone(),
    onError
  });
  const remove = useMutation({
    mutationFn: (id: string) => profileApi.deletePhoto(id),
    onSuccess: () => onDone(),
    onError
  });
  const reorder = useMutation({
    mutationFn: (ids: string[]) => profileApi.reorderPhotos(ids),
    onSuccess: (profile) => onDone(profile),
    onError
  });

  const openMenu = (photo: Photo, index: number) => {
    const actions: { text: string; style?: "destructive" | "cancel"; onPress?: () => void }[] = [];
    if (index > 0) {
      actions.push({
        text: "Make main photo",
        onPress: () =>
          reorder.mutate([photo.id, ...photos.filter((x) => x.id !== photo.id).map((x) => x.id)])
      });
    }
    actions.push({ text: "Replace", onPress: () => replace.mutate(photo.id) });
    if (photos.length > 1) {
      actions.push({
        text: "Delete",
        style: "destructive",
        onPress: () => remove.mutate(photo.id)
      });
    }
    actions.push({ text: "Cancel", style: "cancel" });
    Alert.alert("Photo", undefined, actions);
  };

  const busy = add.isPending || replace.isPending || remove.isPending || reorder.isPending;
  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i]);

  return (
    <View style={styles.grid}>
      {slots.map((photo, index) => (
        <Pressable
          key={photo?.id ?? `empty-${index}`}
          testID={photo ? `photo-${index}` : `photo-add-${index}`}
          accessibilityRole="button"
          accessibilityLabel={photo ? `Photo ${index + 1}, open options` : "Add a photo"}
          disabled={busy || (!photo && index !== photos.length)}
          onPress={() => (photo ? openMenu(photo, index) : add.mutate())}
          style={[
            styles.slot,
            { borderColor: p.border, backgroundColor: p.surface, opacity: busy ? 0.6 : 1 }
          ]}
        >
          {photo ? (
            <>
              <AuthImage path={photo.url} style={styles.image} />
              {index === 0 ? (
                <View style={[styles.badge, { backgroundColor: p.primary }]}>
                  <Text variant="caption" style={{ color: p.primaryText, fontWeight: "700" }}>
                    Main
                  </Text>
                </View>
              ) : null}
            </>
          ) : (
            <Text variant="heading" tone={index === photos.length ? "primary" : "muted"}>
              +
            </Text>
          )}
        </Pressable>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  slot: {
    width: "31%",
    aspectRatio: 4 / 5,
    borderWidth: 1,
    borderStyle: "dashed",
    borderRadius: radii.md,
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden"
  },
  image: { width: "100%", height: "100%" },
  badge: {
    position: "absolute",
    left: 6,
    bottom: 6,
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 2
  }
});
