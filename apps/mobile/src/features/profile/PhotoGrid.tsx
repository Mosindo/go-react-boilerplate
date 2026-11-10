import React, { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { errorMessage, mediaSource } from "../../api/client";
import { photosApi } from "../../api/endpoints";
import type { Photo } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Sheet, Text, showToast } from "../../ui";
import { pickAndCompressPhoto } from "../../utils/pickPhoto";

const SLOTS = 6;

type PhotoGridProps = {
  photos: Photo[];
  onChange: (photos: Photo[]) => void;
};

export function PhotoGrid({ photos, onChange }: PhotoGridProps) {
  const theme = useTheme();
  const [busySlot, setBusySlot] = useState<number | null>(null);
  const [selected, setSelected] = useState<Photo | null>(null);

  const run = async (slot: number, task: () => Promise<Photo[] | null>) => {
    setBusySlot(slot);
    try {
      const next = await task();
      if (next) onChange(next);
    } catch (e) {
      showToast(e instanceof Error && !("status" in e) ? e.message : errorMessage(e), "error");
    } finally {
      setBusySlot(null);
    }
  };

  const add = (slot: number) =>
    run(slot, async () => {
      const uri = await pickAndCompressPhoto();
      return uri ? photosApi.upload(uri) : null;
    });

  const act = async (action: "main" | "replace" | "delete") => {
    const photo = selected;
    setSelected(null);
    if (!photo) return;
    const slot = photos.findIndex((p) => p.id === photo.id);
    if (action === "main") {
      await run(slot, () => photosApi.reorder([photo.id, ...photos.filter((p) => p.id !== photo.id).map((p) => p.id)]));
    } else if (action === "replace") {
      await run(slot, async () => {
        const uri = await pickAndCompressPhoto();
        return uri ? photosApi.replace(photo.id, uri) : null;
      });
    } else {
      await run(slot, () => photosApi.remove(photo.id));
    }
  };

  return (
    <View>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }).map((_, i) => {
          const photo = photos[i];
          const busy = busySlot === i;
          return (
            <Pressable
              accessibilityLabel={photo ? `Photo ${i + 1}${i === 0 ? ", principale" : ""}. Modifier` : "Ajouter une photo"}
              accessibilityRole="button"
              disabled={busySlot !== null || (!photo && i !== photos.length)}
              key={photo?.id ?? `empty-${i}`}
              onPress={() => (photo ? setSelected(photo) : add(i))}
              style={[
                styles.slot,
                { borderRadius: theme.radii.md, backgroundColor: theme.colors.surfaceAlt, borderColor: theme.colors.border },
                !photo && styles.dashed
              ]}
              testID={photo ? `photo-slot-${i}` : i === photos.length ? "photo-add" : undefined}
            >
              {photo ? (
                <Image cachePolicy="memory-disk" contentFit="cover" source={mediaSource(photo)} style={StyleSheet.absoluteFill} transition={150} />
              ) : (
                <Ionicons color={i === photos.length ? theme.colors.primary : theme.colors.textSubtle} name="add" size={28} />
              )}
              {i === 0 && photo ? (
                <View style={[styles.badge, { backgroundColor: theme.colors.primary }]}>
                  <Text tone="onPrimary" variant="caption">
                    Principale
                  </Text>
                </View>
              ) : null}
              {busy ? (
                <View style={[StyleSheet.absoluteFill, styles.busy, { backgroundColor: theme.colors.overlay }]}>
                  <ActivityIndicator color="#fff" />
                </View>
              ) : null}
            </Pressable>
          );
        })}
      </View>
      <Text style={styles.hint} tone="subtle" variant="caption">
        Jusqu’à 6 photos. La première est votre photo principale. Les métadonnées (dont la position GPS) sont supprimées à l’envoi.
      </Text>
      <Sheet onClose={() => setSelected(null)} title="Cette photo" visible={!!selected}>
        <View style={styles.actions}>
          {selected && photos[0]?.id !== selected.id ? <Button label="Définir comme principale" onPress={() => act("main")} variant="secondary" /> : null}
          <Button label="Remplacer" onPress={() => act("replace")} variant="secondary" />
          <Button label="Supprimer" onPress={() => act("delete")} variant="danger" />
        </View>
      </Sheet>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  slot: { width: "31%", aspectRatio: 4 / 5, overflow: "hidden", alignItems: "center", justifyContent: "center", borderWidth: 1, flexGrow: 1, maxWidth: "32%" },
  dashed: { borderStyle: "dashed" },
  badge: { position: "absolute", left: 6, bottom: 6, paddingHorizontal: 8, paddingVertical: 2, borderRadius: 999 },
  busy: { alignItems: "center", justifyContent: "center" },
  hint: { marginTop: 10 },
  actions: { gap: 10, paddingVertical: 8 }
});
