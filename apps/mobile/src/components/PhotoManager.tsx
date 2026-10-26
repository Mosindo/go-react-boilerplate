import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { photosApi, type Photo } from "../api";
import { pickImage } from "../lib/pickImage";
import { queryKeys } from "../lib/queryKeys";
import { radius, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { useFeedback } from "./Feedback";
import { Photo as PhotoView } from "./Photo";
import { Text } from "./Text";

const MAX_PHOTOS = 6;

type Props = { photos: Photo[] };

/** Grid of up to 6 photos: add, replace, delete and choose the main picture. */
export function PhotoManager({ photos }: Props) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const { toast, confirm } = useFeedback();
  const [busyId, setBusyId] = useState<string | null>(null);

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.profile });
  const fail = (err: unknown) => toast(err instanceof Error ? err.message : "Action impossible.", "error");

  const add = useMutation({
    mutationFn: async () => {
      const image = await pickImage();
      if (image) await photosApi.add(image);
    },
    onSuccess: refresh,
    onError: fail
  });

  const replace = useMutation({
    mutationFn: async (id: string) => {
      const image = await pickImage();
      if (image) await photosApi.replace(id, image);
    },
    onSuccess: refresh,
    onError: fail,
    onSettled: () => setBusyId(null)
  });

  const remove = useMutation({
    mutationFn: (id: string) => photosApi.remove(id),
    onSuccess: refresh,
    onError: fail,
    onSettled: () => setBusyId(null)
  });

  const makeMain = useMutation({
    mutationFn: (id: string) =>
      photosApi.reorder([id, ...photos.filter((p) => p.id !== id).map((p) => p.id)]),
    onSuccess: refresh,
    onError: fail,
    onSettled: () => setBusyId(null)
  });

  const onDelete = async (photo: Photo) => {
    const ok = await confirm({
      title: "Supprimer cette photo ?",
      message: photos.length === 1 ? "Sans photo, votre profil ne sera plus visible." : undefined,
      confirmLabel: "Supprimer",
      destructive: true
    });
    if (ok) {
      setBusyId(photo.id);
      remove.mutate(photo.id);
    }
  };

  return (
    <View style={styles.grid}>
      {photos.map((photo, i) => (
        <View key={photo.id} style={[styles.cell, { borderColor: i === 0 ? theme.primary : theme.border }]}>
          <PhotoView url={photo.url} label={`Photo ${i + 1}`} style={styles.image} />
          {i === 0 ? (
            <View style={[styles.badge, { backgroundColor: theme.primary }]}>
              <Text variant="caption" color={theme.onPrimary}>
                Principale
              </Text>
            </View>
          ) : null}
          <View style={styles.cellActions}>
            {i !== 0 ? (
              <Button
                label="Principale"
                variant="ghost"
                small
                disabled={busyId === photo.id}
                onPress={() => {
                  setBusyId(photo.id);
                  makeMain.mutate(photo.id);
                }}
              />
            ) : null}
            <Button
              label="Changer"
              variant="ghost"
              small
              disabled={busyId === photo.id}
              onPress={() => {
                setBusyId(photo.id);
                replace.mutate(photo.id);
              }}
            />
            <Button
              label="Retirer"
              variant="ghost"
              small
              disabled={busyId === photo.id}
              onPress={() => void onDelete(photo)}
            />
          </View>
        </View>
      ))}
      {photos.length < MAX_PHOTOS ? (
        <View style={[styles.cell, styles.addCell, { borderColor: theme.border }]}>
          <Button
            label="Ajouter une photo"
            variant="secondary"
            small
            loading={add.isPending}
            onPress={() => add.mutate()}
            testID="add-photo"
          />
          <Text variant="caption" muted center>
            {photos.length}/{MAX_PHOTOS} · JPEG ou PNG
          </Text>
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.md },
  cell: { width: "47%", flexGrow: 1, borderWidth: 2, borderRadius: radius.md, overflow: "hidden" },
  image: { width: "100%", aspectRatio: 4 / 5 },
  badge: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.sm,
    paddingHorizontal: spacing.sm,
    borderRadius: radius.pill
  },
  cellActions: { flexDirection: "row", flexWrap: "wrap", justifyContent: "center" },
  addCell: {
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.sm,
    aspectRatio: 4 / 5,
    borderStyle: "dashed",
    padding: spacing.md
  }
});
