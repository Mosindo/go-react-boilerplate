import { useState } from "react";
import { Alert, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { photosApi } from "../api";
import { errorMessage } from "../lib/errors";
import { keys, usePhotos } from "../state/queries";
import { radius, spacing, useColors } from "../theme";
import { useToast } from "./feedback";
import { Loading, PhotoImage, Text } from "./ui";

const SLOTS = 6;

/** Gestion des photos: ajout, remplacement, suppression, photo principale et ordre. */
export function PhotoManager() {
  const colors = useColors();
  const toast = useToast();
  const queryClient = useQueryClient();
  const photos = usePhotos();
  const [busy, setBusy] = useState(false);

  const refresh = () => queryClient.invalidateQueries({ queryKey: keys.photos });

  const pick = async (replaceId?: string) => {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      toast.show("Autorisez l'accès à vos photos dans les réglages.", "error");
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"],
      quality: 0.8,
      allowsEditing: true,
      aspect: [3, 4],
    });
    if (result.canceled || result.assets.length === 0) {
      return;
    }
    const asset = result.assets[0];
    setBusy(true);
    try {
      await photosApi.upload(
        { uri: asset.uri, name: asset.fileName ?? "photo.jpg", type: asset.mimeType ?? "image/jpeg" },
        replaceId,
      );
      await refresh();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(false);
    }
  };

  const mutate = useMutation({
    mutationFn: (action: () => Promise<unknown>) => action(),
    onSuccess: refresh,
    onError: (error) => toast.show(errorMessage(error), "error"),
  });

  if (photos.isLoading) {
    return <Loading />;
  }
  const items = photos.data ?? [];
  const ids = items.map((p) => p.id);

  const openMenu = (index: number) => {
    const photo = items[index];
    const actions = [
      ...(index > 0
        ? [
            {
              text: "Définir comme photo principale",
              onPress: () => mutate.mutate(() => photosApi.makePrimary(photo.id)),
            },
            {
              text: "Déplacer vers la gauche",
              onPress: () => {
                const next = [...ids];
                [next[index - 1], next[index]] = [next[index], next[index - 1]];
                mutate.mutate(() => photosApi.reorder(next));
              },
            },
          ]
        : []),
      { text: "Remplacer", onPress: () => void pick(photo.id) },
      {
        text: "Supprimer",
        style: "destructive" as const,
        onPress: () => mutate.mutate(() => photosApi.remove(photo.id)),
      },
      { text: "Annuler", style: "cancel" as const },
    ];
    Alert.alert("Photo", undefined, actions);
  };

  return (
    <View>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }, (_, index) => {
          const photo = items[index];
          return (
            <Pressable
              accessibilityLabel={photo ? `Photo ${index + 1}` : "Ajouter une photo"}
              accessibilityRole="button"
              disabled={busy || mutate.isPending}
              key={photo?.id ?? `slot-${index}`}
              onPress={() => (photo ? openMenu(index) : void pick())}
              style={[styles.slot, { borderColor: colors.border, backgroundColor: colors.surface }]}
            >
              {photo ? (
                <>
                  <PhotoImage photoId={photo.id} style={StyleSheet.absoluteFill} />
                  {index === 0 ? (
                    <View style={[styles.mainTag, { backgroundColor: colors.primary }]}>
                      <Text style={{ color: colors.onPrimary }} variant="caption">
                        Principale
                      </Text>
                    </View>
                  ) : null}
                </>
              ) : (
                <Ionicons color={colors.textMuted} name="add" size={32} />
              )}
            </Pressable>
          );
        })}
      </View>
      <Text muted style={{ marginTop: spacing.md }} variant="caption">
        {busy ? "Envoi en cours…" : "Jusqu'à 6 photos (JPEG, PNG ou WebP). La première est affichée en premier."}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", justifyContent: "space-between" },
  slot: {
    width: "31.5%",
    aspectRatio: 3 / 4,
    borderRadius: radius.md,
    borderWidth: 1,
    borderStyle: "dashed",
    marginBottom: spacing.md,
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden",
  },
  mainTag: {
    position: "absolute",
    left: 6,
    bottom: 6,
    borderRadius: radius.pill,
    paddingHorizontal: 8,
    paddingVertical: 2,
  },
});
