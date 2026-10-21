import { Ionicons } from "@expo/vector-icons";
import { Image } from "expo-image";
import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Photo } from "../api/types";
import { absoluteUrl } from "../api/client";
import { radius, spacing, useTheme } from "../theme";
import { ActionSheet, Loader, Text } from "../shared/ui";

const SLOTS = 6;

type Props = {
  photos: Photo[];
  busy?: boolean;
  onAdd: () => void;
  onReplace: (photo: Photo) => void;
  onDelete: (photo: Photo) => void;
  onMakeMain: (photo: Photo) => void;
  onMove: (photo: Photo, direction: -1 | 1) => void;
};

/** 3 x 2 grid. The first slot is the main photo shown first on the profile card. */
export function PhotoGrid({ photos, busy, onAdd, onReplace, onDelete, onMakeMain, onMove }: Props) {
  const theme = useTheme();
  const [selected, setSelected] = useState<Photo | null>(null);

  const actions = selected
    ? [
        ...(selected.position > 0
          ? [{ label: "Définir comme photo principale", onPress: () => onMakeMain(selected), testID: "photo-action-main" }]
          : []),
        ...(selected.position > 0 ? [{ label: "Déplacer vers la gauche", onPress: () => onMove(selected, -1) }] : []),
        ...(selected.position < photos.length - 1 ? [{ label: "Déplacer vers la droite", onPress: () => onMove(selected, 1) }] : []),
        { label: "Remplacer", onPress: () => onReplace(selected), testID: "photo-action-replace" },
        { label: "Supprimer", destructive: true, onPress: () => onDelete(selected), testID: "photo-action-delete" }
      ]
    : [];

  return (
    <View>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }, (_, i) => {
          const photo = photos[i];
          return (
            <View key={i} style={styles.cell}>
              {photo ? (
                <Pressable
                  testID={`photo-slot-${i}`}
                  accessibilityRole="button"
                  accessibilityLabel={`Photo ${i + 1}${i === 0 ? " (principale)" : ""}, options`}
                  onPress={() => setSelected(photo)}
                  style={[styles.fill, { borderColor: theme.border }]}
                >
                  <Image source={{ uri: absoluteUrl(photo.thumbUrl) }} style={styles.fill} contentFit="cover" transition={120} />
                  {i === 0 ? (
                    <View style={[styles.tag, { backgroundColor: theme.primary }]}>
                      <Text variant="caption" style={{ color: theme.primaryText, fontSize: 11, fontWeight: "700" }}>
                        Principale
                      </Text>
                    </View>
                  ) : null}
                </Pressable>
              ) : i === photos.length ? (
                <Pressable
                  testID="photo-add"
                  accessibilityRole="button"
                  accessibilityLabel="Ajouter une photo"
                  disabled={busy}
                  onPress={onAdd}
                  style={[styles.fill, styles.empty, { borderColor: theme.primary, backgroundColor: theme.primarySoft }]}
                >
                  {busy ? <Loader /> : <Ionicons name="add" size={32} color={theme.primary} />}
                </Pressable>
              ) : (
                <View style={[styles.fill, styles.empty, { borderColor: theme.border, borderStyle: "dashed" }]} />
              )}
            </View>
          );
        })}
      </View>
      <ActionSheet visible={selected !== null} title="Photo" actions={actions} onClose={() => setSelected(null)} />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  cell: { width: "31.5%", aspectRatio: 3 / 4 },
  fill: { flex: 1, borderRadius: radius.md, overflow: "hidden", borderWidth: 1 },
  empty: { alignItems: "center", justifyContent: "center", borderWidth: 1.5 },
  tag: { position: "absolute", left: 6, bottom: 6, borderRadius: radius.pill, paddingHorizontal: 8, paddingVertical: 2 }
});
