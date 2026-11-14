import React, { useState } from "react";
import { ActivityIndicator, Platform, Pressable, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { photosApi } from "../../api/endpoints";
import type { Photo } from "../../api/types";
import { useSetMyProfile } from "../../state/queries";
import { radii, spacing, type Palette } from "../../theme/palette";
import { useStyles, useTheme } from "../../theme/ThemeProvider";
import { Button, Icon, InlineError, RemotePhoto, Sheet, Text } from "../../ui";
import { pickPhoto } from "./media";

const SLOTS = 6;

/** 6-slot photo manager: add, replace, delete, reorder, set main photo. */
export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const setMyProfile = useSetMyProfile();
  const [selected, setSelected] = useState<Photo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busySlot, setBusySlot] = useState<number | null>(null);
  const [sourceSheet, setSourceSheet] = useState<{ replace?: Photo } | null>(null);

  const apply = (next: Photo[]) => setMyProfile((prev) => ({ ...prev, photos: next, complete: !!prev.profile && next.length > 0 }));

  const mutation = useMutation({
    mutationFn: async (op: () => Promise<Photo[]>) => op(),
    onSuccess: (next) => {
      setError(null);
      apply(next);
    },
    onError: (e) => setError(errorMessage(e)),
    onSettled: () => setBusySlot(null)
  });

  const choose = async (source: "library" | "camera", replace?: Photo) => {
    setSourceSheet(null);
    try {
      const uri = await pickPhoto(source);
      if (!uri) return;
      setBusySlot(replace ? replace.position : photos.length);
      mutation.mutate(async () => {
        if (replace) return (await photosApi.replace(replace.id, uri)).photos;
        const created = await photosApi.upload(uri);
        return [...photos, created];
      });
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const startAdd = (replace?: Photo) => {
    setSelected(null);
    if (Platform.OS === "web") {
      void choose("library", replace);
    } else {
      setSourceSheet({ replace });
    }
  };

  const move = (photo: Photo, to: number) => {
    const ids = photos.map((p) => p.id).filter((id) => id !== photo.id);
    ids.splice(to, 0, photo.id);
    setSelected(null);
    setBusySlot(to);
    mutation.mutate(async () => (await photosApi.reorder(ids)).photos);
  };

  const remove = (photo: Photo) => {
    setSelected(null);
    setBusySlot(photo.position);
    mutation.mutate(async () => (await photosApi.remove(photo.id)).photos);
  };

  return (
    <View style={styles.wrap}>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }).map((_, index) => {
          const photo = photos[index];
          const busy = busySlot === index && mutation.isPending;
          return (
            <Pressable
              accessibilityLabel={photo ? `Photo ${index + 1}${index === 0 ? ", principale" : ""}. Options` : "Ajouter une photo"}
              accessibilityRole="button"
              disabled={mutation.isPending || (!photo && index !== photos.length)}
              key={photo?.id ?? `empty-${index}`}
              onPress={() => (photo ? setSelected(photo) : startAdd())}
              style={[styles.slot, !photo ? styles.empty : null]}
              testID={photo ? `photo-slot-${index}` : index === photos.length ? "photo-add" : undefined}
            >
              {photo ? <RemotePhoto photo={photo} style={StyleSheet.absoluteFill} /> : null}
              {!photo && index === photos.length && !busy ? <Icon color={colors.primary} name="add" size={30} /> : null}
              {photo && index === 0 ? (
                <View style={styles.mainBadge}>
                  <Text tone="onPrimary" variant="caption" weight="700">
                    Principale
                  </Text>
                </View>
              ) : null}
              {busy ? (
                <View style={styles.busy}>
                  <ActivityIndicator color="#fff" />
                </View>
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <InlineError message={error} />
      <Text tone="subtle" variant="caption">
        Jusqu’à 6 photos (JPEG, PNG ou WebP, 10 Mo max). Les métadonnées (dont la position GPS) sont supprimées automatiquement.
      </Text>

      <Sheet onClose={() => setSelected(null)} title="Cette photo" visible={!!selected}>
        {selected ? (
          <View style={styles.actions}>
            {selected.position > 0 ? <Button icon="star-outline" label="Définir comme principale" onPress={() => move(selected, 0)} variant="outline" /> : null}
            {selected.position > 0 ? <Button icon="arrow-back" label="Déplacer avant" onPress={() => move(selected, selected.position - 1)} variant="outline" /> : null}
            {selected.position < photos.length - 1 ? (
              <Button icon="arrow-forward" label="Déplacer après" onPress={() => move(selected, selected.position + 1)} variant="outline" />
            ) : null}
            <Button icon="swap-horizontal" label="Remplacer" onPress={() => startAdd(selected)} variant="outline" />
            <Button disabled={photos.length <= 1} icon="trash-outline" label="Supprimer" onPress={() => remove(selected)} variant="danger" />
            {photos.length <= 1 ? (
              <Text tone="subtle" variant="caption">
                Gardez au moins une photo : remplacez-la plutôt.
              </Text>
            ) : null}
          </View>
        ) : null}
      </Sheet>

      <Sheet onClose={() => setSourceSheet(null)} title="Ajouter une photo" visible={!!sourceSheet}>
        <View style={styles.actions}>
          <Button icon="images-outline" label="Choisir dans la galerie" onPress={() => choose("library", sourceSheet?.replace)} variant="outline" />
          <Button icon="camera-outline" label="Prendre une photo" onPress={() => choose("camera", sourceSheet?.replace)} variant="outline" />
        </View>
      </Sheet>
    </View>
  );
}

const makeStyles = (c: Palette) =>
  StyleSheet.create({
    wrap: { gap: spacing.sm },
    grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
    slot: {
      width: "31.8%",
      aspectRatio: 4 / 5,
      borderRadius: radii.md,
      overflow: "hidden",
      backgroundColor: c.surfaceMuted,
      alignItems: "center",
      justifyContent: "center"
    },
    empty: { borderWidth: 1.5, borderStyle: "dashed", borderColor: c.borderStrong, backgroundColor: c.surface },
    mainBadge: { position: "absolute", left: 6, bottom: 6, backgroundColor: c.primary, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 },
    busy: { ...StyleSheet.absoluteFillObject, backgroundColor: "rgba(0,0,0,0.35)", alignItems: "center", justifyContent: "center" },
    actions: { gap: spacing.xs }
  });
