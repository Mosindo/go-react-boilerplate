import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { photosApi, type UploadFile } from "../api/endpoints";
import type { PhotoRef } from "../api/types";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import { radii, useTheme } from "../theme";
import { Dialog, PhotoImage, Text, useToast } from "../ui";

const MAX = 6;

async function pick(): Promise<UploadFile | null> {
  const res = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    quality: 0.8,
    allowsEditing: true,
    aspect: [3, 4]
  });
  const asset = res.canceled ? undefined : res.assets[0];
  if (!asset) return null;
  const isPng = asset.mimeType === "image/png";
  return {
    uri: asset.uri,
    name: asset.fileName ?? (isPng ? "photo.png" : "photo.jpg"),
    type: asset.mimeType ?? "image/jpeg"
  };
}

export function PhotosEditor({ photos }: { photos: PhotoRef[] }) {
  const t = useTheme();
  const qc = useQueryClient();
  const toast = useToast();
  const [selected, setSelected] = useState<PhotoRef | null>(null);

  const refresh = () => qc.invalidateQueries({ queryKey: queryKeys.overview });
  const onError = (err: unknown) => toast.show(errorMessage(err), "error");

  const add = useMutation({
    mutationFn: async () => {
      const file = await pick();
      if (file) await photosApi.upload(file);
    },
    onSuccess: refresh,
    onError
  });
  const replace = useMutation({
    mutationFn: async (id: string) => {
      const file = await pick();
      if (file) await photosApi.replace(id, file);
    },
    onSuccess: () => {
      setSelected(null);
      return refresh();
    },
    onError
  });
  const remove = useMutation({
    mutationFn: (id: string) => photosApi.remove(id),
    onSuccess: () => {
      setSelected(null);
      return refresh();
    },
    onError
  });
  const reorder = useMutation({
    mutationFn: (ids: string[]) => photosApi.reorder(ids),
    onSuccess: () => {
      setSelected(null);
      return refresh();
    },
    onError
  });

  const makeMain = (id: string) =>
    reorder.mutate([id, ...photos.filter((p) => p.id !== id).map((p) => p.id)]);
  const busy = add.isPending || replace.isPending || remove.isPending || reorder.isPending;

  return (
    <View>
      <View style={styles.grid}>
        {Array.from({ length: MAX }, (_, i) => {
          const photo = photos[i];
          return photo ? (
            <Pressable
              key={photo.id}
              accessibilityRole="button"
              accessibilityLabel={
                i === 0 ? "Photo principale, modifier" : `Photo ${i + 1}, modifier`
              }
              onPress={() => setSelected(photo)}
              style={styles.cell}
            >
              <PhotoImage photoId={photo.id} style={styles.fill} />
              {i === 0 ? (
                <View style={[styles.badge, { backgroundColor: t.primary }]}>
                  <Text variant="caption" tone="onPrimary" style={styles.badgeText}>
                    Principale
                  </Text>
                </View>
              ) : null}
            </Pressable>
          ) : (
            <Pressable
              key={`empty-${i}`}
              accessibilityRole="button"
              accessibilityLabel="Ajouter une photo"
              disabled={busy || i !== photos.length}
              onPress={() => add.mutate()}
              style={[
                styles.cell,
                styles.empty,
                {
                  borderColor: t.border,
                  backgroundColor: t.surface,
                  opacity: i === photos.length ? 1 : 0.45
                }
              ]}
            >
              <Ionicons
                name={add.isPending && i === photos.length ? "hourglass-outline" : "add"}
                size={28}
                color={t.primary}
              />
            </Pressable>
          );
        })}
      </View>
      <Text variant="caption" tone="muted" style={styles.hint}>
        Jusqu'à {MAX} photos (JPEG ou PNG). La première est votre photo principale.
      </Text>

      <Dialog visible={!!selected} title="Cette photo" onCancel={() => setSelected(null)}>
        <View style={styles.menu}>
          {selected && photos[0]?.id !== selected.id ? (
            <MenuItem
              icon="star-outline"
              label="Définir comme principale"
              onPress={() => makeMain(selected.id)}
            />
          ) : null}
          {selected ? (
            <MenuItem
              icon="swap-horizontal-outline"
              label="Remplacer"
              onPress={() => replace.mutate(selected.id)}
            />
          ) : null}
          {selected ? (
            <MenuItem
              icon="trash-outline"
              label="Supprimer"
              danger
              onPress={() => remove.mutate(selected.id)}
            />
          ) : null}
        </View>
      </Dialog>
    </View>
  );
}

function MenuItem({
  icon,
  label,
  onPress,
  danger
}: {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  onPress: () => void;
  danger?: boolean;
}) {
  const t = useTheme();
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={styles.menuItem}>
      <Ionicons name={icon} size={20} color={danger ? t.danger : t.text} />
      <Text tone={danger ? "danger" : "default"}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  cell: { width: "31%", aspectRatio: 3 / 4, borderRadius: radii.md, overflow: "hidden" },
  fill: { flex: 1 },
  empty: {
    borderWidth: 1.5,
    borderStyle: "dashed",
    alignItems: "center",
    justifyContent: "center"
  },
  badge: {
    position: "absolute",
    left: 6,
    bottom: 6,
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 2
  },
  badgeText: { fontWeight: "700", fontSize: 11 },
  hint: { marginTop: 10 },
  menu: { gap: 4 },
  menuItem: { flexDirection: "row", alignItems: "center", gap: 12, paddingVertical: 12 }
});
