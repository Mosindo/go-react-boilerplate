import React, { useState } from "react";
import { ActivityIndicator, Image, Pressable, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { absoluteUrl } from "../api/client";
import { friendlyError } from "../api/errors";
import { deletePhoto, makeMainPhoto, replacePhoto, reorderPhotos, uploadPhoto } from "../api/platform";
import type { PhotoRef } from "../api/types";
import { queryKeys } from "../hooks/queries";
import { pickPhoto, type PickResult } from "../lib/media";
import { Notice, Text, colors, radii, spacing } from "../shared/ui";
import { ActionSheet } from "./ActionSheet";

export const MAX_PHOTOS = 6;

type PhotoManagerProps = {
  photos: PhotoRef[];
};

async function pickOrExplain(): Promise<{ result: PickResult; message: string | null }> {
  const result = await pickPhoto();
  return {
    result,
    message: result.status === "denied" ? "Autorisez l'accès à vos photos dans les réglages de votre téléphone." : null
  };
}

export function PhotoManager({ photos }: PhotoManagerProps) {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [selected, setSelected] = useState<PhotoRef | null>(null);

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.me });

  async function run(task: () => Promise<void>) {
    setBusy(true);
    setMessage(null);
    try {
      await task();
      await refresh();
    } catch (error) {
      setMessage(friendlyError(error));
    } finally {
      setBusy(false);
    }
  }

  const add = () =>
    run(async () => {
      const { result, message: denied } = await pickOrExplain();
      if (denied) {
        setMessage(denied);
      }
      if (result.status === "picked") {
        await uploadPhoto(result.image);
      }
    });

  const replace = (photo: PhotoRef) =>
    run(async () => {
      const { result, message: denied } = await pickOrExplain();
      if (denied) {
        setMessage(denied);
      }
      if (result.status === "picked") {
        await replacePhoto(photo.id, result.image);
      }
    });

  const move = (photo: PhotoRef, delta: -1 | 1) =>
    run(async () => {
      const ids = photos.map((p) => p.id);
      const from = ids.indexOf(photo.id);
      const to = from + delta;
      if (from < 0 || to < 0 || to >= ids.length) {
        return;
      }
      [ids[from], ids[to]] = [ids[to], ids[from]];
      await reorderPhotos(ids);
    });

  const index = selected ? photos.findIndex((p) => p.id === selected.id) : -1;
  const actions = selected
    ? [
        ...(index > 0
          ? [
              { label: "Définir comme photo principale", onPress: () => void run(() => makeMainPhoto(selected.id)) },
              { label: "Déplacer vers l'avant", onPress: () => void move(selected, -1) }
            ]
          : []),
        ...(index < photos.length - 1 ? [{ label: "Déplacer vers l'arrière", onPress: () => void move(selected, 1) }] : []),
        { label: "Remplacer la photo", onPress: () => void replace(selected) },
        { label: "Supprimer la photo", destructive: true, onPress: () => void run(() => deletePhoto(selected.id)) }
      ]
    : [];

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);

  return (
    <View style={styles.wrap}>
      <View style={styles.grid}>
        {slots.map((photo, i) => (
          <View key={photo?.id ?? `empty-${i}`} style={styles.cell}>
            {photo ? (
              <Pressable
                accessibilityLabel={`Photo ${i + 1}${i === 0 ? ", photo principale" : ""}. Modifier`}
                accessibilityRole="button"
                disabled={busy}
                onPress={() => setSelected(photo)}
                style={styles.slot}
                testID={`photo-${i}`}
              >
                <Image source={{ uri: absoluteUrl(photo.url) }} style={styles.image} />
                {i === 0 ? (
                  <View style={styles.mainBadge}>
                    <Text tone="inverse" variant="caption" weight="bold">
                      Principale
                    </Text>
                  </View>
                ) : null}
              </Pressable>
            ) : (
              <Pressable
                accessibilityLabel="Ajouter une photo"
                accessibilityRole="button"
                disabled={busy || i !== photos.length}
                onPress={() => void add()}
                style={[styles.slot, styles.empty, i !== photos.length ? styles.emptyDim : null]}
                testID={i === photos.length ? "add-photo" : undefined}
              >
                <Text tone="primary" variant="heading" weight="bold">
                  +
                </Text>
              </Pressable>
            )}
          </View>
        ))}
        {busy ? (
          <View style={styles.busy}>
            <ActivityIndicator color={colors.primary} />
          </View>
        ) : null}
      </View>
      <Text tone="muted" variant="caption">
        La première photo est celle que l'on voit en premier. Les images sont recadrées et nettoyées (métadonnées supprimées) avant
        publication.
      </Text>
      {message ? <Notice title={message} tone="danger" /> : null}
      <ActionSheet actions={actions} onClose={() => setSelected(null)} title="Cette photo" visible={selected !== null} />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 4 / 5 },
  slot: { flex: 1, borderRadius: radii.lg, overflow: "hidden", alignItems: "center", justifyContent: "center" },
  image: { width: "100%", height: "100%" },
  empty: { borderWidth: 1.5, borderStyle: "dashed", borderColor: colors.primaryBorder, backgroundColor: colors.primarySoft },
  emptyDim: { opacity: 0.45 },
  mainBadge: {
    position: "absolute",
    left: spacing.xs,
    bottom: spacing.xs,
    backgroundColor: colors.scrim,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
    borderRadius: radii.pill
  },
  busy: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: "rgba(250,246,241,0.6)",
    alignItems: "center",
    justifyContent: "center",
    borderRadius: radii.lg
  }
});
