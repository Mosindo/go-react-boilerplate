import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import type { Photo } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { radii, spacing, useTheme } from "../../theme";
import { ActionSheet, Icon, PhotoImage, Text, toast } from "../../ui";
import { usePhotoMutations } from "./hooks";
import { pickPhoto } from "./media";

const SLOTS = 6;

export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const { colors } = useTheme();
  const { upload, replace, remove, reorder } = usePhotoMutations();
  const [selected, setSelected] = useState<Photo | null>(null);
  const busy = upload.isPending || replace.isPending || remove.isPending || reorder.isPending;

  const run = async (action: () => Promise<unknown>, success?: string) => {
    try {
      await action();
      if (success) toast(success, "success");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const add = async () => {
    const picked = await pickPhoto();
    if (!picked) return;
    if ("error" in picked) {
      toast(picked.error, "error");
      return;
    }
    await run(() => upload.mutateAsync(picked.uri));
  };

  const makePrimary = (photo: Photo) => {
    const ids = [photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)];
    void run(() => reorder.mutateAsync(ids), "Photo principale mise à jour");
  };

  const swap = async (photo: Photo) => {
    const picked = await pickPhoto();
    if (!picked) return;
    if ("error" in picked) {
      toast(picked.error, "error");
      return;
    }
    await run(() => replace.mutateAsync({ id: photo.id, uri: picked.uri }), "Photo remplacée");
  };

  const slots = Array.from({ length: SLOTS }, (_, i) => photos[i]);
  return (
    <View>
      <View style={styles.grid}>
        {slots.map((photo, index) => {
          const isNext = !photo && index === photos.length;
          return (
            <Pressable
              key={photo?.id ?? `empty-${index}`}
              accessibilityRole="button"
              accessibilityLabel={photo ? `Photo ${index + 1}${index === 0 ? ", principale" : ""}. Options` : "Ajouter une photo"}
              disabled={busy || (!photo && !isNext)}
              onPress={() => (photo ? setSelected(photo) : add())}
              testID={photo ? `photo-slot-${index}` : isNext ? "photo-add" : undefined}
              style={[
                styles.slot,
                index === 0 && styles.primarySlot,
                { backgroundColor: colors.surfaceAlt, borderColor: isNext ? colors.primary : colors.border },
                !photo && styles.dashed
              ]}
            >
              {photo ? (
                <>
                  <PhotoImage photo={photo} style={StyleSheet.absoluteFill} />
                  {index === 0 ? (
                    <View style={[styles.tag, { backgroundColor: colors.primary }]}>
                      <Text variant="micro" color="onPrimary">
                        PRINCIPALE
                      </Text>
                    </View>
                  ) : null}
                </>
              ) : isNext ? (
                busy ? (
                  <ActivityIndicator color={colors.primary} />
                ) : (
                  <Icon name="add" size={28} color="primary" />
                )
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <Text variant="caption" color="textSubtle" style={styles.hint}>
        Jusqu’à 6 photos. Les métadonnées (dont la position GPS) sont supprimées à l’envoi.
      </Text>
      <ActionSheet
        visible={Boolean(selected)}
        onClose={() => setSelected(null)}
        title="Photo"
        actions={
          selected
            ? [
                ...(selected.position !== 0 ? [{ label: "Définir comme principale", onPress: () => makePrimary(selected) }] : []),
                { label: "Remplacer", onPress: () => void swap(selected) },
                ...(photos.length > 1
                  ? [
                      {
                        label: "Supprimer",
                        destructive: true,
                        onPress: () => void run(() => remove.mutateAsync(selected.id), "Photo supprimée")
                      }
                    ]
                  : [])
              ]
            : []
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  slot: {
    width: "31.8%",
    aspectRatio: 3 / 4,
    borderRadius: radii.md,
    borderWidth: 1.5,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  primarySlot: {},
  dashed: { borderStyle: "dashed" },
  tag: { position: "absolute", left: 6, bottom: 6, borderRadius: radii.pill, paddingHorizontal: 7, paddingVertical: 3 },
  hint: { marginTop: spacing.xs }
});
