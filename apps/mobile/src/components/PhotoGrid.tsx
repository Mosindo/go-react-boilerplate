import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage, mediaUrl } from "../api/client";
import { deletePhoto, reorderPhotos, replacePhoto, uploadPhoto } from "../api/dating";
import type { Photo } from "../api/types";
import { qk } from "../hooks/queries";
import { confirm } from "../lib/confirm";
import { pickPhotoForUpload } from "../lib/media";
import { Text, colors, radii, showToast, spacing } from "../ui";

const SLOTS = 6;

type Props = {
  photos: Photo[];
};

/**
 * Six-slot photo editor: add, replace, delete, set as main and move left.
 * The first photo is the main one shown in discovery.
 */
export function PhotoGrid({ photos }: Props) {
  const client = useQueryClient();
  const [busy, setBusy] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);

  const run = async (key: string, action: () => Promise<Photo[] | null>) => {
    setBusy(key);
    try {
      const next = await action();
      if (next) {
        await client.invalidateQueries({ queryKey: qk.profile });
      }
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setBusy(null);
    }
  };

  const add = () =>
    run("add", async () => {
      const form = await pickPhotoForUpload();
      return form ? uploadPhoto(form) : null;
    });

  const replace = (photo: Photo) =>
    run(photo.id, async () => {
      const form = await pickPhotoForUpload();
      return form ? replacePhoto(photo.id, form) : null;
    });

  const remove = async (photo: Photo) => {
    const warning = photos.length === 1 ? "Sans photo, votre profil ne sera plus visible." : "Cette photo sera définitivement supprimée.";
    if (await confirm("Supprimer la photo ?", warning, "Supprimer", true)) {
      setSelected(null);
      await run(photo.id, () => deletePhoto(photo.id));
    }
  };

  const move = (photo: Photo, toIndex: number) =>
    run(photo.id, () => {
      const ids = photos.map((p) => p.id).filter((id) => id !== photo.id);
      ids.splice(toIndex, 0, photo.id);
      return reorderPhotos(ids);
    });

  const active = photos.find((p) => p.id === selected) ?? null;

  return (
    <View style={styles.wrap}>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }).map((_, index) => {
          const photo = photos[index];
          if (!photo) {
            const isNext = index === photos.length;
            return (
              <Pressable
                accessibilityLabel="Ajouter une photo"
                accessibilityRole="button"
                disabled={!isNext || busy !== null}
                key={`empty-${index}`}
                onPress={add}
                style={[styles.slot, styles.empty, !isNext ? styles.dim : null]}
                testID={isNext ? "photo-add" : undefined}
              >
                {busy === "add" && isNext ? (
                  <ActivityIndicator color={colors.accent} />
                ) : (
                  <Ionicons color={isNext ? colors.accent : colors.textSubtle} name="add" size={28} />
                )}
              </Pressable>
            );
          }
          return (
            <Pressable
              accessibilityHint="Ouvre les options de cette photo"
              accessibilityLabel={index === 0 ? "Photo principale" : `Photo ${index + 1}`}
              key={photo.id}
              onPress={() => setSelected(selected === photo.id ? null : photo.id)}
              style={[styles.slot, selected === photo.id ? styles.selected : null]}
              testID={`photo-${index}`}
            >
              <Image cachePolicy="memory-disk" contentFit="cover" source={{ uri: mediaUrl(photo.url) }} style={StyleSheet.absoluteFill} />
              {index === 0 ? (
                <View style={styles.badge}>
                  <Text tone="onAccent" variant="eyebrow">
                    Principale
                  </Text>
                </View>
              ) : null}
              {busy === photo.id ? (
                <View style={styles.busy}>
                  <ActivityIndicator color={colors.text} />
                </View>
              ) : null}
            </Pressable>
          );
        })}
      </View>
      {active ? (
        <View style={styles.actions}>
          {photos.indexOf(active) > 0 ? <ActionButton icon="star-outline" label="Principale" onPress={() => move(active, 0)} /> : null}
          {photos.indexOf(active) > 1 ? (
            <ActionButton icon="arrow-back" label="Avancer" onPress={() => move(active, photos.indexOf(active) - 1)} />
          ) : null}
          <ActionButton icon="swap-horizontal" label="Remplacer" onPress={() => replace(active)} />
          <ActionButton danger icon="trash-outline" label="Supprimer" onPress={() => remove(active)} />
        </View>
      ) : (
        <Text tone="subtle" variant="caption">
          Touchez une photo pour la remplacer, la déplacer ou la supprimer. JPEG ou PNG, 6 maximum.
        </Text>
      )}
    </View>
  );
}

function ActionButton({
  icon,
  label,
  onPress,
  danger
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  danger?: boolean;
}) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={({ pressed }) => [styles.action, pressed ? styles.dim : null]}>
      <Ionicons color={danger ? colors.danger : colors.text} name={icon} size={18} />
      <Text tone={danger ? "danger" : "default"} variant="caption">
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.sm
  },
  grid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  },
  slot: {
    width: "31.8%",
    aspectRatio: 4 / 5,
    borderRadius: radii.md,
    overflow: "hidden",
    backgroundColor: colors.surfaceRaised
  },
  empty: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1.5,
    borderStyle: "dashed",
    borderColor: colors.borderStrong
  },
  dim: {
    opacity: 0.45
  },
  selected: {
    borderWidth: 2,
    borderColor: colors.accent
  },
  badge: {
    position: "absolute",
    left: 6,
    bottom: 6,
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: radii.sm,
    backgroundColor: colors.accent
  },
  busy: {
    ...StyleSheet.absoluteFillObject,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.overlay
  },
  actions: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  },
  action: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderRadius: radii.pill,
    backgroundColor: colors.surfaceRaised
  }
});
