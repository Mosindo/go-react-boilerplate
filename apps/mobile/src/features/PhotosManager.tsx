import React, { useState } from "react";
import { ActivityIndicator, Modal, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { errorMessage } from "../api/client";
import { profileApi } from "../api/endpoints";
import type { Photo as PhotoType } from "../api/types";
import { useProfileMutation } from "../hooks/queries";
import { useTheme } from "../theme/ThemeProvider";
import { Button, Photo, Text, useFeedback } from "../ui";
import { pickPhoto } from "./media";

const SLOTS = 6;

/** 3×2 photo grid: add, replace, delete and choose the main photo. */
export function PhotosManager({ photos }: { photos: PhotoType[] }) {
  const { colors, radii } = useTheme();
  const { toast, confirm } = useFeedback();
  const [selected, setSelected] = useState<PhotoType | null>(null);
  const [busy, setBusy] = useState(false);

  const upload = useProfileMutation(profileApi.uploadPhoto);
  const replace = useProfileMutation(
    ({ id, image }: { id: string; image: Parameters<typeof profileApi.uploadPhoto>[0] }) =>
      profileApi.replacePhoto(id, image)
  );
  const remove = useProfileMutation(profileApi.deletePhoto);
  const reorder = useProfileMutation(profileApi.reorderPhotos);

  const run = async (task: () => Promise<unknown>, success?: string) => {
    setBusy(true);
    try {
      await task();
      if (success) {
        toast(success, "success");
      }
    } catch (error) {
      toast(errorMessage(error, error instanceof Error ? error.message : "Action impossible."), "error");
    } finally {
      setBusy(false);
    }
  };

  const add = () =>
    run(async () => {
      const image = await pickPhoto();
      if (image) {
        await upload.mutateAsync(image);
      }
    });

  const close = () => setSelected(null);

  return (
    <View style={{ gap: 10 }}>
      <View style={styles.grid} testID="photos-grid">
        {Array.from({ length: SLOTS }).map((_, i) => {
          const photo = photos[i];
          return (
            <Pressable
              key={photo?.id ?? `empty-${i}`}
              testID={photo ? `photo-slot-${i}` : i === photos.length ? "photo-add" : undefined}
              accessibilityRole="button"
              accessibilityLabel={photo ? `Photo ${i + 1}, options` : "Ajouter une photo"}
              disabled={busy || (!photo && i !== photos.length)}
              onPress={() => (photo ? setSelected(photo) : add())}
              style={[
                styles.slot,
                {
                  borderRadius: radii.md,
                  borderColor: photo ? "transparent" : colors.borderStrong,
                  backgroundColor: colors.surfaceSunken
                }
              ]}
            >
              {photo ? (
                <>
                  <Photo url={photo.url} rounded={radii.md} style={StyleSheet.absoluteFillObject} />
                  {i === 0 ? (
                    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                      <Text variant="caption" style={{ color: colors.onPrimary }} weight="700">
                        Principale
                      </Text>
                    </View>
                  ) : null}
                </>
              ) : i === photos.length ? (
                busy ? (
                  <ActivityIndicator color={colors.primary} />
                ) : (
                  <Ionicons name="add" size={30} color={colors.primary} />
                )
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <Text variant="caption" tone="subtle">
        JPEG, PNG ou WebP, 10 Mo max. Les métadonnées (dont la position GPS) sont supprimées automatiquement.
      </Text>

      <Modal transparent visible={!!selected} animationType="fade" onRequestClose={close}>
        <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={close}>
          <View style={[styles.sheet, { backgroundColor: colors.surfaceRaised, borderRadius: radii.lg }]}>
            <Text variant="heading">Photo</Text>
            {selected && photos[0]?.id !== selected.id ? (
              <Button
                label="Définir comme principale"
                variant="secondary"
                icon="star-outline"
                onPress={() => {
                  const target = selected;
                  close();
                  void run(() =>
                    reorder.mutateAsync([target.id, ...photos.filter((p) => p.id !== target.id).map((p) => p.id)])
                  );
                }}
              />
            ) : null}
            <Button
              label="Remplacer"
              variant="secondary"
              icon="swap-horizontal"
              onPress={() => {
                const target = selected;
                close();
                if (target) {
                  void run(async () => {
                    const image = await pickPhoto();
                    if (image) {
                      await replace.mutateAsync({ id: target.id, image });
                    }
                  }, "Photo remplacée");
                }
              }}
            />
            <Button
              testID="photo-delete"
              label="Supprimer"
              variant="danger"
              icon="trash-outline"
              onPress={async () => {
                const target = selected;
                close();
                if (!target) {
                  return;
                }
                const ok = await confirm({
                  title: "Supprimer cette photo ?",
                  message:
                    photos.length === 1 ? "Sans photo, ton profil ne sera plus visible dans la découverte." : undefined,
                  confirmLabel: "Supprimer",
                  destructive: true
                });
                if (ok) {
                  void run(() => remove.mutateAsync(target.id), "Photo supprimée");
                }
              }}
            />
          </View>
        </Pressable>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  slot: {
    width: "31%",
    aspectRatio: 3 / 4,
    borderWidth: 1.5,
    borderStyle: "dashed",
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden"
  },
  badge: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 8, paddingVertical: 3, borderRadius: 999 },
  backdrop: { flex: 1, justifyContent: "flex-end", padding: 16 },
  sheet: { padding: 18, gap: 10, width: "100%", maxWidth: 520, alignSelf: "center" }
});
