import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as ImagePicker from "expo-image-picker";
import { useQueryClient } from "@tanstack/react-query";
import { describeError } from "../api/client";
import { keys } from "../api/keys";
import * as api from "../api/platform";
import type { Me, Photo } from "../api/types";
import { showToast } from "../shared/feedback";
import { Loader, PhotoImage, Sheet, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";
import { confirm } from "../utils/confirm";

const MAX_PHOTOS = 6;

type PhotoManagerProps = { photos: Photo[] };

/** Grid of up to six photos: add, replace, make main, remove. Position 0 is the main photo. */
export function PhotoManager({ photos }: PhotoManagerProps) {
  const { colors } = useTheme();
  const client = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<Photo | null>(null);

  const apply = (next: Photo[]) => {
    client.setQueryData<Me>(keys.me, (me) =>
      me ? { ...me, photos: next, onboarding: { ...me.onboarding, hasPhoto: next.length > 0, complete: me.onboarding.hasProfile && next.length > 0 } } : me
    );
    void client.invalidateQueries({ queryKey: keys.discover });
  };

  const run = async (task: () => Promise<Photo[]>) => {
    setBusy(true);
    try {
      apply(await task());
    } catch (error) {
      showToast(describeError(error), "error");
    } finally {
      setBusy(false);
    }
  };

  const pick = async (): Promise<ImagePicker.ImagePickerAsset | null> => {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      showToast("Autorisez l'accès à vos photos dans les réglages pour en ajouter.", "error");
      return null;
    }
    const result = await ImagePicker.launchImageLibraryAsync({ mediaTypes: ["images"], quality: 0.85, allowsEditing: true, aspect: [3, 4] });
    return result.canceled ? null : (result.assets[0] ?? null);
  };

  const add = async () => {
    const asset = await pick();
    if (asset) await run(() => api.uploadPhoto({ uri: asset.uri, mimeType: asset.mimeType, fileName: asset.fileName }));
  };

  const replace = async (photo: Photo) => {
    const asset = await pick();
    if (asset) await run(() => api.replacePhoto(photo.id, { uri: asset.uri, mimeType: asset.mimeType, fileName: asset.fileName }));
  };

  const makeMain = (photo: Photo) => run(() => api.reorderPhotos([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]));

  const remove = async (photo: Photo) => {
    if (await confirm({ title: "Supprimer cette photo ?", confirmLabel: "Supprimer", destructive: true })) {
      await run(() => api.deletePhoto(photo.id));
    }
  };

  const slots = Array.from({ length: MAX_PHOTOS }, (_, index) => photos[index] ?? null);

  return (
    <View>
      <View style={styles.grid}>
        {slots.map((photo, index) => (
          <View key={photo?.id ?? `empty-${index}`} style={styles.cell}>
            {photo ? (
              <Pressable
                accessibilityLabel={index === 0 ? "Photo principale, modifier" : `Photo ${index + 1}, modifier`}
                accessibilityRole="button"
                onPress={() => setSelected(photo)}
                style={styles.fill}
              >
                <PhotoImage path={photo.thumbUrl} style={styles.fill} />
                {index === 0 ? (
                  <View style={[styles.mainTag, { backgroundColor: colors.primary }]}>
                    <Text style={{ color: colors.onPrimary }} variant="caption" weight="bold">
                      Principale
                    </Text>
                  </View>
                ) : null}
              </Pressable>
            ) : (
              <Pressable
                accessibilityLabel="Ajouter une photo"
                accessibilityRole="button"
                disabled={busy || index !== photos.length}
                onPress={add}
                style={[
                  styles.fill,
                  styles.empty,
                  { borderColor: colors.border, backgroundColor: colors.surfaceMuted, opacity: index === photos.length ? 1 : 0.45 }
                ]}
              >
                <Ionicons color={colors.primary} name="add" size={30} />
              </Pressable>
            )}
          </View>
        ))}
      </View>
      {busy ? (
        <View style={styles.busy}>
          <Loader />
          <Text tone="muted" variant="bodySmall">
            Envoi en cours…
          </Text>
        </View>
      ) : null}
      <Sheet
        actions={
          selected
            ? [
                ...(selected.id !== photos[0]?.id
                  ? [{ label: "Définir comme photo principale", icon: "star-outline" as const, onPress: () => void makeMain(selected) }]
                  : []),
                { label: "Remplacer", icon: "swap-horizontal-outline", onPress: () => void replace(selected) },
                { label: "Supprimer", icon: "trash-outline", destructive: true, onPress: () => void remove(selected) }
              ]
            : []
        }
        onClose={() => setSelected(null)}
        title="Cette photo"
        visible={selected !== null}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 3 / 4 },
  fill: { flex: 1, borderRadius: radii.md, overflow: "hidden" },
  empty: { alignItems: "center", justifyContent: "center", borderWidth: 1.5, borderStyle: "dashed" },
  mainTag: { position: "absolute", left: 6, bottom: 6, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 },
  busy: { flexDirection: "row", gap: spacing.sm, alignItems: "center", marginTop: spacing.md }
});
