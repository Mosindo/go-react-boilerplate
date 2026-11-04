import React, { useState } from "react";
import * as ImagePicker from "expo-image-picker";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { photosApi } from "../../api/endpoints";
import type { OwnProfile, Photo } from "../../api/types";
import { keys } from "../../hooks/queries";
import { confirm } from "../../shared/feedback/confirm";
import { Button, RemoteImage, Sheet, Text, radius, spacing, useColors, useToast } from "../../shared/ui";

export const MAX_PHOTOS = 6;

async function pickImage(): Promise<{ uri: string; mimeType?: string } | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) throw new Error("Autorisez l'accès à vos photos dans les réglages de l'appareil.");
  const result = await ImagePicker.launchImageLibraryAsync({ mediaTypes: ["images"], quality: 0.8, allowsEditing: false });
  const asset = result.canceled ? undefined : result.assets[0];
  return asset ? { uri: asset.uri, mimeType: asset.mimeType ?? undefined } : null;
}

export function PhotosManager({ profile }: { profile: OwnProfile }) {
  const qc = useQueryClient();
  const toast = useToast();
  const c = useColors();
  const [selected, setSelected] = useState<Photo | null>(null);
  const photos = profile.photos;

  const refresh = () => qc.invalidateQueries({ queryKey: keys.profile });
  const fail = (err: unknown) => toast.show(err instanceof Error && !("status" in err) ? err.message : errorMessage(err), "error");

  const add = useMutation({
    mutationFn: async () => {
      const file = await pickImage();
      if (file) await photosApi.upload(file.uri, file.mimeType);
    },
    onSuccess: refresh,
    onError: fail,
  });
  const replace = useMutation({
    mutationFn: async (id: string) => {
      const file = await pickImage();
      if (file) await photosApi.replace(id, file.uri, file.mimeType);
    },
    onSuccess: refresh,
    onError: fail,
  });
  const remove = useMutation({ mutationFn: (id: string) => photosApi.remove(id), onSuccess: refresh, onError: fail });
  const primary = useMutation({ mutationFn: (id: string) => photosApi.makePrimary(id), onSuccess: refresh, onError: fail });
  const reorder = useMutation({ mutationFn: (ids: string[]) => photosApi.reorder(ids), onSuccess: refresh, onError: fail });

  const move = (photo: Photo, delta: -1 | 1) => {
    const ids = photos.map((p) => p.id);
    const from = ids.indexOf(photo.id);
    const to = from + delta;
    if (to < 0 || to >= ids.length) return;
    const [item] = ids.splice(from, 1);
    if (item) ids.splice(to, 0, item);
    reorder.mutate(ids);
    setSelected(null);
  };

  const busy = add.isPending || replace.isPending || remove.isPending || primary.isPending || reorder.isPending;
  const index = selected ? photos.findIndex((p) => p.id === selected.id) : -1;

  return (
    <View style={{ gap: spacing.md }}>
      <View style={styles.grid}>
        {photos.map((p, i) => (
          <Pressable key={p.id} accessibilityRole="button" accessibilityLabel={`Photo ${i + 1}${i === 0 ? ", photo principale" : ""}. Ouvrir les actions`} onPress={() => setSelected(p)} style={styles.cell} testID={`photo-${i}`}>
            <RemoteImage path={p.url} style={[styles.img, { borderColor: i === 0 ? c.accent : c.border }]} />
            {i === 0 ? <View style={[styles.tag, { backgroundColor: c.accent }]}><Text variant="caption" style={{ color: c.onAccent }}>Principale</Text></View> : null}
          </Pressable>
        ))}
        {photos.length < MAX_PHOTOS ? (
          <Pressable accessibilityRole="button" accessibilityLabel="Ajouter une photo" onPress={() => add.mutate()} disabled={busy} style={[styles.cell, styles.add, { borderColor: c.border, backgroundColor: c.surface }]} testID="photo-add">
            <Text variant="heading" tone="primary">{add.isPending ? "…" : "+"}</Text>
            <Text variant="caption" tone="muted">Ajouter</Text>
          </Pressable>
        ) : null}
      </View>
      <Text variant="caption" tone="muted">{photos.length}/{MAX_PHOTOS} photos. Les photos sont recompressées et nettoyées de leurs métadonnées (dont la position GPS).</Text>

      <Sheet visible={selected !== null} title="Cette photo" onClose={() => setSelected(null)}>
        {selected ? (
          <>
            {index > 0 ? <Button title="Définir comme photo principale" onPress={() => { primary.mutate(selected.id); setSelected(null); }} /> : null}
            {index > 0 ? <Button title="Déplacer vers le début" variant="secondary" onPress={() => move(selected, -1)} /> : null}
            {index < photos.length - 1 ? <Button title="Déplacer vers la fin" variant="secondary" onPress={() => move(selected, 1)} /> : null}
            <Button title="Remplacer" variant="secondary" onPress={() => { replace.mutate(selected.id); setSelected(null); }} />
            <Button title="Supprimer" variant="danger" onPress={() => { const id = selected.id; setSelected(null); confirm("Supprimer la photo ?", "Cette action est définitive.", "Supprimer", () => remove.mutate(id)); }} />
          </>
        ) : null}
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 3 / 4 },
  img: { flex: 1, borderRadius: radius.md, borderWidth: 2 },
  add: { borderWidth: 1.5, borderStyle: "dashed", borderRadius: radius.md, alignItems: "center", justifyContent: "center" },
  tag: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 8, paddingVertical: 2, borderRadius: radius.pill },
});
