import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { photoApi, type UploadFile } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import type { Photo } from "../api/types";
import { PhotoImage } from "../shared/media/PhotoImage";
import { Loader, Notice, Text, radii, spacing, useTheme } from "../shared/ui";
import { ActionSheet, type SheetAction } from "./ActionSheet";

export const MAX_PHOTOS = 6;
const MAX_BYTES = 5 * 1024 * 1024;

type PhotoManagerProps = {
  /** Called after any change so the parent can refresh onboarding status. */
  onChanged?: (photos: Photo[]) => void;
};

async function pickPhoto(): Promise<UploadFile | { error: string } | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 0.8
  });
  if (result.canceled || !result.assets[0]) {
    return null;
  }
  const asset = result.assets[0];
  if (asset.fileSize && asset.fileSize > MAX_BYTES) {
    return { error: "Cette photo dépasse 5 Mo. Choisissez-en une plus légère." };
  }
  const type = asset.mimeType === "image/png" ? "image/png" : "image/jpeg";
  return { uri: asset.uri, name: type === "image/png" ? "photo.png" : "photo.jpg", type };
}

export function PhotoManager({ onChanged }: PhotoManagerProps) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Photo | null>(null);
  const photos = useQuery({ queryKey: queryKeys.photos, queryFn: photoApi.list });

  const refresh = async (next?: Photo[]) => {
    const list = next ?? (await photoApi.list());
    queryClient.setQueryData(queryKeys.photos, list);
    void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
    onChanged?.(list);
  };

  const fail = (e: unknown) => setError(e instanceof ApiError ? e.message : "Une erreur est survenue.");

  const add = useMutation({
    mutationFn: photoApi.add,
    onSuccess: () => refresh(),
    onError: fail
  });
  const replace = useMutation({
    mutationFn: ({ id, file }: { id: string; file: UploadFile }) => photoApi.replace(id, file),
    onSuccess: () => refresh(),
    onError: fail
  });
  const remove = useMutation({ mutationFn: photoApi.remove, onSuccess: () => refresh(), onError: fail });
  const reorder = useMutation({ mutationFn: photoApi.reorder, onSuccess: (list) => refresh(list), onError: fail });

  const busy = add.isPending || replace.isPending || remove.isPending || reorder.isPending;
  const list = photos.data ?? [];

  const upload = async (then: (file: UploadFile) => void) => {
    setError(null);
    const picked = await pickPhoto();
    if (!picked) {
      return;
    }
    if ("error" in picked) {
      setError(picked.error);
      return;
    }
    then(picked);
  };

  const move = (photo: Photo, delta: number) => {
    const ids = list.map((p) => p.id);
    const from = ids.indexOf(photo.id);
    const to = from + delta;
    if (to < 0 || to >= ids.length) {
      return;
    }
    ids.splice(to, 0, ids.splice(from, 1)[0] as string);
    reorder.mutate(ids);
  };

  const actionsFor = (photo: Photo): SheetAction[] => {
    const index = list.findIndex((p) => p.id === photo.id);
    const actions: SheetAction[] = [];
    if (index > 0) {
      actions.push({ label: "Définir comme photo principale", onPress: () => move(photo, -index) });
      actions.push({ label: "Déplacer vers la gauche", onPress: () => move(photo, -1) });
    }
    if (index < list.length - 1) {
      actions.push({ label: "Déplacer vers la droite", onPress: () => move(photo, 1) });
    }
    actions.push({ label: "Remplacer la photo", onPress: () => void upload((file) => replace.mutate({ id: photo.id, file })) });
    actions.push({ label: "Supprimer", destructive: true, onPress: () => remove.mutate(photo.id) });
    return actions;
  };

  if (photos.isLoading) {
    return <Loader />;
  }

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => list[i]);

  return (
    <View style={styles.root}>
      <View style={styles.grid}>
        {slots.map((photo, i) =>
          photo ? (
            <Pressable
              accessibilityHint="Ouvre les actions : déplacer, remplacer, supprimer"
              accessibilityLabel={i === 0 ? "Photo principale" : `Photo ${i + 1}`}
              accessibilityRole="button"
              disabled={busy}
              key={photo.id}
              onPress={() => setSelected(photo)}
              style={[styles.slot, { borderColor: i === 0 ? colors.primary : colors.border }]}
            >
              <PhotoImage path={photo.url} style={styles.image} />
              {i === 0 ? (
                <View style={[styles.primaryTag, { backgroundColor: colors.primary }]}>
                  <Text style={{ color: colors.primaryForeground }} variant="caption" weight="bold">
                    Principale
                  </Text>
                </View>
              ) : null}
            </Pressable>
          ) : (
            <Pressable
              accessibilityLabel="Ajouter une photo"
              accessibilityRole="button"
              disabled={busy || i !== list.length}
              key={`empty-${i}`}
              onPress={() => void upload((file) => add.mutate(file))}
              style={[styles.slot, styles.empty, { borderColor: colors.borderStrong, opacity: i === list.length ? 1 : 0.45 }]}
            >
              {busy && i === list.length ? <Loader /> : <Ionicons color={colors.textMuted} name="add" size={28} />}
            </Pressable>
          )
        )}
      </View>
      <Text tone="muted" variant="caption">
        {list.length}/{MAX_PHOTOS} photos · JPEG ou PNG, 5 Mo max. La première est votre photo principale.
      </Text>
      {error ? <Notice message={error} tone="danger" /> : null}
      <ActionSheet
        actions={selected ? actionsFor(selected) : []}
        onClose={() => setSelected(null)}
        title="Cette photo"
        visible={selected !== null}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.sm },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: { width: "31.5%", aspectRatio: 3 / 4, borderRadius: radii.md, borderWidth: 2, overflow: "hidden" },
  image: { width: "100%", height: "100%" },
  empty: { borderStyle: "dashed", alignItems: "center", justifyContent: "center" },
  primaryTag: { position: "absolute", left: 6, bottom: 6, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 }
});
