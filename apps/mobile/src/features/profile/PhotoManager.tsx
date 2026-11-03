import React, { useState } from "react";
import { Platform, Pressable, StyleSheet, Text, View } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api";
import { errorMessage } from "../../api/client";
import type { PhotoRef } from "../../api/types";
import { radius, spacing, useTheme } from "../../theme";
import { AuthImage } from "../../ui/AuthImage";
import { Button, ErrorText } from "../../ui/components";

const MAX_PHOTOS = 6;

async function buildForm(asset: ImagePicker.ImagePickerAsset): Promise<FormData> {
  const form = new FormData();
  const name = asset.fileName ?? "photo.jpg";
  const type = asset.mimeType ?? "image/jpeg";
  if (Platform.OS === "web") {
    const blob = await (await fetch(asset.uri)).blob();
    form.append("file", blob, name);
  } else {
    form.append("file", { uri: asset.uri, name, type } as unknown as Blob);
  }
  return form;
}

async function pickImage(): Promise<ImagePicker.ImagePickerAsset | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.8
  });
  return result.canceled ? null : (result.assets[0] ?? null);
}

export function PhotoManager({ photos }: { photos: PhotoRef[] }) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const refresh = () => queryClient.invalidateQueries({ queryKey: ["profile"] });
  const run = useMutation({
    mutationFn: async (task: () => Promise<unknown>) => task(),
    onSuccess: () => {
      setError(null);
      return refresh();
    },
    onError: (e) => setError(errorMessage(e))
  });

  const add = async () => {
    const asset = await pickImage();
    if (asset) {
      run.mutate(async () => api.uploadPhoto(await buildForm(asset)));
    }
  };
  const replace = async (id: string) => {
    const asset = await pickImage();
    if (asset) {
      run.mutate(async () => api.replacePhoto(id, await buildForm(asset)));
    }
  };
  const move = (index: number, delta: -1 | 1) => {
    const ids = photos.map((p) => p.id);
    const target = index + delta;
    if (target < 0 || target >= ids.length) {
      return;
    }
    [ids[index], ids[target]] = [ids[target], ids[index]];
    run.mutate(() => api.reorderPhotos(ids));
  };

  return (
    <View style={{ gap: spacing.md }}>
      <ErrorText message={error} />
      <View style={styles.grid}>
        {photos.map((photo, index) => (
          <View key={photo.id} style={styles.cell} testID={`photo-${index}`}>
            <AuthImage path={photo.url} label={`Photo ${index + 1}`} style={styles.image} />
            {index === 0 ? (
              <View style={[styles.main, { backgroundColor: theme.primary }]}>
                <Text style={{ color: theme.primaryText, fontSize: 11, fontWeight: "700" }}>Principale</Text>
              </View>
            ) : null}
            <View style={styles.actions}>
              <Tool label="←" hint="Déplacer avant" onPress={() => move(index, -1)} disabled={index === 0} />
              <Tool label="→" hint="Déplacer après" onPress={() => move(index, 1)} disabled={index === photos.length - 1} />
              <Tool label="↻" hint="Remplacer" onPress={() => void replace(photo.id)} />
              <Tool label="✕" hint="Supprimer" onPress={() => run.mutate(() => api.deletePhoto(photo.id))} />
            </View>
          </View>
        ))}
      </View>
      {photos.length < MAX_PHOTOS ? (
        <Button
          title={photos.length === 0 ? "Ajouter ma première photo" : "Ajouter une photo"}
          onPress={() => void add()}
          variant="secondary"
          loading={run.isPending}
          testID="add-photo"
        />
      ) : null}
      <Text
        style={{ color: theme.textMuted, fontSize: 13 }}
      >{`${photos.length}/${MAX_PHOTOS} photos · JPEG, PNG ou WebP · 8 Mo maximum`}</Text>
    </View>
  );
}

function Tool({ label, hint, onPress, disabled }: { label: string; hint: string; onPress: () => void; disabled?: boolean }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={hint}
      disabled={disabled}
      onPress={onPress}
      style={[styles.tool, { opacity: disabled ? 0.3 : 1 }]}
    >
      <Text style={styles.toolText}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.md },
  cell: { width: "31%", aspectRatio: 0.8, borderRadius: radius.md, overflow: "hidden" },
  image: { width: "100%", height: "100%" },
  main: { position: "absolute", top: 6, left: 6, borderRadius: radius.pill, paddingHorizontal: 8, paddingVertical: 2 },
  actions: { position: "absolute", bottom: 0, left: 0, right: 0, flexDirection: "row", backgroundColor: "rgba(0,0,0,0.55)" },
  tool: { flex: 1, minHeight: 36, alignItems: "center", justifyContent: "center" },
  toolText: { color: "#FFFFFF", fontSize: 16, fontWeight: "700" }
});
