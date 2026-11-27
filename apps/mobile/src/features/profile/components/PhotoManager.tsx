import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { photosApi } from "../../../api/endpoints";
import { errorMessage } from "../../../api/client";
import type { MyProfile, Photo } from "../../../api/types";
import { keys, queryClient } from "../../../hooks/queryClient";
import { useTheme } from "../../../theme/ThemeProvider";
import { radii, spacing } from "../../../theme/palette";
import { Icon, InlineError, ListRow, RemoteImage, Sheet, Text, toast } from "../../../ui";
import { pickPhoto } from "../../../utils/photoPicker";

const MAX_PHOTOS = 6;

function setPhotos(photos: Photo[]) {
  queryClient.setQueryData<MyProfile>(keys.myProfile, (p) =>
    p ? { ...p, photos, missing: photos.length ? p.missing.filter((m) => m !== "photos") : p.missing, isComplete: p.exists && photos.length > 0 } : p
  );
}

export function PhotoManager({ photos }: { photos: Photo[] }) {
  const { colors } = useTheme();
  const [selected, setSelected] = useState<Photo | null>(null);
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: async (action: { kind: "add" } | { kind: "replace"; id: string } | { kind: "delete"; id: string } | { kind: "main"; id: string }) => {
      setError(null);
      if (action.kind === "delete") {
        return photosApi.remove(action.id);
      }
      if (action.kind === "main") {
        return photosApi.reorder([action.id, ...photos.filter((p) => p.id !== action.id).map((p) => p.id)]);
      }
      const picked = await pickPhoto();
      if (!picked) {
        return null;
      }
      if ("error" in picked) {
        throw new Error(picked.error);
      }
      return action.kind === "add" ? photosApi.upload(picked.photo) : photosApi.replace(action.id, picked.photo);
    },
    onSuccess: (result) => {
      if (result) {
        setPhotos(result);
      }
    },
    onError: (e) => setError(errorMessage(e))
  });

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);
  const firstEmpty = photos.length;

  return (
    <View style={styles.wrap}>
      <View style={styles.grid}>
        {slots.map((photo, index) => {
          if (photo) {
            return (
              <Pressable
                key={photo.id}
                style={styles.slot}
                onPress={() => setSelected(photo)}
                accessibilityRole="button"
                accessibilityLabel={index === 0 ? "Photo principale, options" : `Photo ${index + 1}, options`}
              >
                <RemoteImage url={photo.url} style={[StyleSheet.absoluteFill, styles.image]} />
                {index === 0 ? (
                  <View style={[styles.mainBadge, { backgroundColor: colors.primary }]}>
                    <Text variant="overline" style={{ color: colors.onPrimary }}>
                      PRINCIPALE
                    </Text>
                  </View>
                ) : null}
              </Pressable>
            );
          }
          const isNext = index === firstEmpty;
          return (
            <Pressable
              key={`empty-${index}`}
              disabled={!isNext || mutation.isPending}
              onPress={() => mutation.mutate({ kind: "add" })}
              accessibilityRole="button"
              accessibilityLabel="Ajouter une photo"
              testID={isNext ? "photo-add" : undefined}
              style={[styles.slot, styles.empty, { borderColor: isNext ? colors.primary : colors.border, backgroundColor: colors.surface }]}
            >
              {isNext && mutation.isPending ? (
                <ActivityIndicator color={colors.primary} />
              ) : (
                <Icon name="add" size={28} color={isNext ? colors.primary : colors.borderStrong} />
              )}
            </Pressable>
          );
        })}
      </View>
      <Text variant="caption" tone="subtle">
        JPEG, PNG ou WebP · 10 Mo max · jusqu’à 6 photos. Les métadonnées (dont la position GPS) sont supprimées automatiquement.
      </Text>
      <InlineError message={error} />

      <Sheet visible={!!selected} title="Photo" onClose={() => setSelected(null)}>
        {selected && photos[0]?.id !== selected.id ? (
          <ListRow
            icon="star-outline"
            title="Définir comme photo principale"
            onPress={() => {
              mutation.mutate({ kind: "main", id: selected.id });
              setSelected(null);
            }}
          />
        ) : null}
        <ListRow
          icon="swap-horizontal-outline"
          title="Remplacer"
          onPress={() => {
            if (selected) mutation.mutate({ kind: "replace", id: selected.id });
            setSelected(null);
          }}
        />
        <ListRow
          icon="trash-outline"
          title="Supprimer"
          destructive
          onPress={() => {
            if (!selected) return;
            if (photos.length === 1) {
              toast("Garde au moins une photo : ajoute-en une autre avant de supprimer celle-ci.", "info");
            } else {
              mutation.mutate({ kind: "delete", id: selected.id });
            }
            setSelected(null);
          }}
        />
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.sm },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  slot: { width: "31.5%", aspectRatio: 3 / 4, borderRadius: radii.md, overflow: "hidden", alignItems: "center", justifyContent: "center" },
  image: { borderRadius: radii.md },
  empty: { borderWidth: 1.5, borderStyle: "dashed" },
  mainBadge: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 6, paddingVertical: 2, borderRadius: 6 }
});
