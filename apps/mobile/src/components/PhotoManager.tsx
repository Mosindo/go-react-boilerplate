import React, { useState } from "react";
import { ActivityIndicator, Pressable, View } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { mediaUrl } from "../api/client";
import { deletePhoto, reorderPhotos, replacePhoto, setMainPhoto, uploadPhoto } from "../api/platform";
import type { PhotoView } from "../api/types";
import { errorMessage } from "../i18n/errors";
import { pickPhoto, type PickResult } from "../lib/imagePicker";
import { MAX_PHOTOS, moveId, sortedPhotos } from "../lib/photos";
import { showError } from "../shared/feedback";
import { Badge, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";
import { ActionSheet, type SheetAction } from "./ActionSheet";

type PhotoManagerProps = {
  photos: PhotoView[];
  onChanged: (photos: PhotoView[]) => void;
};

type Sheet = { kind: "photo"; photo: PhotoView; index: number } | { kind: "source"; replaceId?: string } | null;

export function PhotoManager({ onChanged, photos }: PhotoManagerProps) {
  const palette = useTheme();
  const ordered = sortedPhotos(photos);
  const [sheet, setSheet] = useState<Sheet>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const run = async (key: string, task: () => Promise<PhotoView[]>) => {
    setBusy(key);
    try {
      onChanged(await task());
    } catch (error) {
      showError(errorMessage(error, "Impossible de mettre à jour vos photos."));
    } finally {
      setBusy(null);
    }
  };

  const choose = async (source: "library" | "camera", replaceId?: string) => {
    let picked: PickResult;
    try {
      picked = await pickPhoto(source);
    } catch {
      showError("Impossible d'ouvrir vos photos.");
      return;
    }
    if (picked.status === "denied") {
      showError(
        source === "camera" ? "Autorisez l'appareil photo dans les réglages." : "Autorisez l'accès à vos photos dans les réglages."
      );
      return;
    }
    if (picked.status !== "ok") {
      return;
    }
    const { file } = picked;
    await run(replaceId ?? "new", () => (replaceId ? replacePhoto(replaceId, file) : uploadPhoto(file)));
  };

  const photoActions = (photo: PhotoView, index: number): SheetAction[] => {
    const ids = ordered.map((p) => p.id);
    const actions: SheetAction[] = [];
    if (index > 0) {
      actions.push({ label: "Définir comme photo principale", onPress: () => void run(photo.id, () => setMainPhoto(photo.id)) });
      actions.push({
        label: "Déplacer vers la gauche",
        onPress: () => void run(photo.id, () => reorderPhotos(moveId(ids, index, index - 1)))
      });
    }
    if (index < ordered.length - 1) {
      actions.push({
        label: "Déplacer vers la droite",
        onPress: () => void run(photo.id, () => reorderPhotos(moveId(ids, index, index + 1)))
      });
    }
    actions.push({ label: "Remplacer la photo", onPress: () => setSheet({ kind: "source", replaceId: photo.id }) });
    actions.push({
      label: ordered.length === 1 ? "Supprimer (une photo minimum)" : "Supprimer la photo",
      destructive: true,
      disabled: ordered.length === 1,
      onPress: () => void run(photo.id, () => deletePhoto(photo.id))
    });
    return actions;
  };

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => ordered[i] ?? null);

  return (
    <View>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {slots.map((photo, index) => {
          const isBusy = busy !== null && (busy === "new" ? !photo && index === ordered.length : busy === photo?.id);
          return (
            <Pressable
              accessibilityLabel={
                photo ? `Photo ${index + 1}${index === 0 ? ", principale" : ""}. Ouvrir les options` : "Ajouter une photo"
              }
              accessibilityRole="button"
              disabled={busy !== null || (!photo && index !== ordered.length)}
              key={photo?.id ?? `empty-${index}`}
              onPress={() => setSheet(photo ? { kind: "photo", photo, index } : { kind: "source" })}
              style={({ pressed }) => ({
                width: "31.5%",
                aspectRatio: 3 / 4,
                borderRadius: radii.md,
                overflow: "hidden",
                borderWidth: photo ? 0 : 2,
                borderStyle: "dashed",
                borderColor: palette.borderStrong,
                backgroundColor: photo ? palette.surfaceMuted : "transparent",
                alignItems: "center",
                justifyContent: "center",
                opacity: pressed ? 0.8 : !photo && index !== ordered.length ? 0.45 : 1
              })}
              testID={photo ? `photo-slot-${index}` : `photo-add-${index}`}
            >
              {photo ? (
                <Image
                  cachePolicy="memory-disk"
                  contentFit="cover"
                  recyclingKey={photo.id}
                  source={{ uri: mediaUrl(photo.thumbUrl) }}
                  style={{ width: "100%", height: "100%" }}
                />
              ) : (
                <Ionicons color={palette.textSubtle} name="add" size={32} />
              )}
              {index === 0 && photo ? (
                <View style={{ position: "absolute", left: 6, bottom: 6 }}>
                  <Badge label="Principale" tone="primary" />
                </View>
              ) : null}
              {isBusy ? (
                <View
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    backgroundColor: palette.scrim,
                    alignItems: "center",
                    justifyContent: "center"
                  }}
                >
                  <ActivityIndicator color="#FFFFFF" />
                </View>
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <Text style={{ marginTop: spacing.sm }} tone="muted" variant="caption">
        {ordered.length}/{MAX_PHOTOS} photos · la première est affichée en premier sur votre profil.
      </Text>

      <ActionSheet
        actions={sheet?.kind === "photo" ? photoActions(sheet.photo, sheet.index) : []}
        onClose={() => setSheet(null)}
        title="Cette photo"
        visible={sheet?.kind === "photo"}
      />
      <ActionSheet
        actions={[
          {
            label: "Choisir dans la photothèque",
            onPress: () => void choose("library", sheet?.kind === "source" ? sheet.replaceId : undefined),
            testID: "photo-source-library"
          },
          { label: "Prendre une photo", onPress: () => void choose("camera", sheet?.kind === "source" ? sheet.replaceId : undefined) }
        ]}
        onClose={() => setSheet(null)}
        title="Ajouter une photo"
        visible={sheet?.kind === "source"}
      />
    </View>
  );
}
