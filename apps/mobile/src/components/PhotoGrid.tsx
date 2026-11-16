import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { mediaUrl } from "../api/client";
import type { Photo } from "../api/types";
import { usePhotoMutations } from "../hooks/queries";
import { colors, radii, spacing } from "../theme/tokens";
import { Button, Sheet, Text, showToast } from "../ui";
import { errorMessage } from "../utils/errors";
import { PhotoPermissionError, pickPhoto } from "../utils/photoPicker";

const SLOTS = 6;

/**
 * Six-slot photo editor: add, replace, delete and set as main photo.
 * Position 0 is the main photo shown first in discovery.
 */
export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const { upload, replace, remove, reorder } = usePhotoMutations();
  const [selected, setSelected] = useState<Photo | null>(null);
  const busy = upload.isPending || replace.isPending || remove.isPending || reorder.isPending;

  const withPicker = async (action: (p: { uri: string; mimeType?: string }) => Promise<unknown>, success: string) => {
    try {
      const picked = await pickPhoto();
      if (!picked) {
        return;
      }
      await action(picked);
      showToast(success, "success");
    } catch (err) {
      showToast(err instanceof PhotoPermissionError ? err.message : errorMessage(err), "error");
    }
  };

  const add = () => withPicker((p) => upload.mutateAsync(p), "Photo ajoutée");

  const makeMain = async (photo: Photo) => {
    const ids = [photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)];
    try {
      await reorder.mutateAsync(ids);
      showToast("Photo principale mise à jour", "success");
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  const del = async (photo: Photo) => {
    try {
      await remove.mutateAsync(photo.id);
      showToast("Photo supprimée", "success");
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  return (
    <View style={styles.grid}>
      {Array.from({ length: SLOTS }).map((_, i) => {
        const photo = photos[i];
        if (photo) {
          return (
            <Pressable
              key={photo.id}
              style={styles.slot}
              onPress={() => setSelected(photo)}
              accessibilityRole="button"
              accessibilityLabel={i === 0 ? "Photo principale, options" : `Photo ${i + 1}, options`}
              testID={`photo-slot-${i}`}
            >
              <Image source={{ uri: mediaUrl(photo.url) }} style={StyleSheet.absoluteFill} contentFit="cover" />
              {i === 0 ? (
                <View style={styles.mainTag}>
                  <Text variant="label" style={{ color: colors.textOnAccent }}>
                    PRINCIPALE
                  </Text>
                </View>
              ) : null}
            </Pressable>
          );
        }
        const isNext = i === photos.length;
        return (
          <Pressable
            key={`empty-${i}`}
            style={[styles.slot, styles.empty, isNext && styles.next]}
            onPress={isNext && !busy ? add : undefined}
            disabled={!isNext || busy}
            accessibilityRole="button"
            accessibilityLabel="Ajouter une photo"
            testID={isNext ? "photo-add" : undefined}
          >
            {isNext && busy ? <ActivityIndicator color={colors.accent} /> : <Ionicons name="add" size={28} color={isNext ? colors.accent : colors.border} />}
          </Pressable>
        );
      })}

      <Sheet visible={!!selected} title="Photo" onClose={() => setSelected(null)}>
        {selected ? (
          <>
            {photos[0]?.id !== selected.id ? (
              <Button
                label="Définir comme principale"
                icon="star-outline"
                variant="secondary"
                onPress={() => {
                  const p = selected;
                  setSelected(null);
                  void makeMain(p);
                }}
              />
            ) : null}
            <Button
              label="Remplacer"
              icon="swap-horizontal"
              variant="secondary"
              onPress={() => {
                const p = selected;
                setSelected(null);
                void withPicker((picked) => replace.mutateAsync({ id: p.id, photo: picked }), "Photo remplacée");
              }}
            />
            <Button
              label="Supprimer"
              icon="trash-outline"
              variant="danger"
              onPress={() => {
                const p = selected;
                setSelected(null);
                void del(p);
              }}
              testID="photo-delete"
            />
          </>
        ) : null}
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  slot: { width: "31.5%", aspectRatio: 4 / 5, borderRadius: radii.md, overflow: "hidden", backgroundColor: colors.surfaceRaised },
  empty: { borderWidth: 1, borderStyle: "dashed", borderColor: colors.border, alignItems: "center", justifyContent: "center", backgroundColor: colors.surface },
  next: { borderColor: colors.accent },
  mainTag: { position: "absolute", left: 6, bottom: 6, backgroundColor: colors.accent, borderRadius: radii.sm, paddingHorizontal: 6, paddingVertical: 2 }
});
