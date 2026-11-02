import React, { useState } from "react";
import { Alert, Platform, Pressable, Text, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiErrorMessage } from "../api/client";
import { photoApi } from "../api/endpoints";
import type { PhotoRef } from "../api/types";
import { PROFILE_KEY } from "../hooks";
import { spacing, radii, useTheme } from "../theme/theme";
import { pickPhoto } from "./photoPicker";
import { useToast } from "./Toast";
import { ActionSheet, AppText, Loading, PhotoImage } from "./ui";

const MAX = 6;

/** Grid of the user's photos: add, replace, delete, set as main, reorder. */
export function PhotoManager() {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<PhotoRef | null>(null);

  const photos = useQuery({ queryKey: ["photos"], queryFn: photoApi.list });
  const items = photos.data?.items ?? [];

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ["photos"] });
    void queryClient.invalidateQueries({ queryKey: PROFILE_KEY });
  };
  const fail = (e: unknown) => toast.show(apiErrorMessage(e), "error");

  const add = useMutation({
    mutationFn: async () => {
      const file = await pickPhoto();
      if (file) await photoApi.upload(file);
    },
    onSuccess: refresh,
    onError: fail
  });
  const replace = useMutation({
    mutationFn: async (id: string) => {
      const file = await pickPhoto();
      if (file) await photoApi.replace(id, file);
    },
    onSuccess: refresh,
    onError: fail
  });
  const remove = useMutation({ mutationFn: (id: string) => photoApi.remove(id), onSuccess: refresh, onError: fail });
  const setMain = useMutation({ mutationFn: (id: string) => photoApi.setMain(id), onSuccess: refresh, onError: fail });
  const reorder = useMutation({ mutationFn: (ids: string[]) => photoApi.reorder(ids), onSuccess: refresh, onError: fail });

  const move = (index: number, delta: number) => {
    const ids = items.map((p) => p.id);
    const target = index + delta;
    if (target < 0 || target >= ids.length) return;
    const a = ids[index];
    const b = ids[target];
    if (a === undefined || b === undefined) return;
    ids[index] = b;
    ids[target] = a;
    reorder.mutate(ids);
  };

  const confirmDelete = (photo: PhotoRef) => {
    if (items.length <= 1) {
      toast.show("Gardez au moins une photo : remplacez-la plutôt.", "error");
      return;
    }
    if (Platform.OS === "web") {
      remove.mutate(photo.id);
      return;
    }
    Alert.alert("Supprimer cette photo ?", undefined, [
      { text: "Annuler", style: "cancel" },
      { text: "Supprimer", style: "destructive", onPress: () => remove.mutate(photo.id) }
    ]);
  };

  if (photos.isLoading) return <Loading />;
  const busy = add.isPending || replace.isPending || remove.isPending || setMain.isPending || reorder.isPending;
  const index = selected ? items.findIndex((p) => p.id === selected.id) : -1;

  return (
    <View style={{ gap: spacing.md }}>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {items.map((photo, i) => (
          <Pressable
            key={photo.id}
            accessibilityRole="button"
            accessibilityLabel={i === 0 ? "Photo principale, options" : `Photo ${i + 1}, options`}
            onPress={() => setSelected(photo)}
            style={{ width: "31%", aspectRatio: 0.8 }}
          >
            <PhotoImage url={photo.url} style={{ width: "100%", height: "100%", borderRadius: radii.md }} />
            {i === 0 ? (
              <View
                style={{
                  position: "absolute",
                  left: 6,
                  bottom: 6,
                  backgroundColor: colors.primary,
                  borderRadius: radii.pill,
                  paddingHorizontal: 8,
                  paddingVertical: 2
                }}
              >
                <Text style={{ color: colors.primaryText, fontSize: 11, fontWeight: "800" }}>Principale</Text>
              </View>
            ) : null}
          </Pressable>
        ))}
        {items.length < MAX ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Ajouter une photo"
            onPress={() => add.mutate()}
            disabled={busy}
            testID="add-photo"
            style={{
              width: "31%",
              aspectRatio: 0.8,
              borderRadius: radii.md,
              borderWidth: 2,
              borderStyle: "dashed",
              borderColor: colors.border,
              alignItems: "center",
              justifyContent: "center"
            }}
          >
            <Text style={{ color: colors.primary, fontSize: 32 }}>+</Text>
          </Pressable>
        ) : null}
      </View>
      <AppText variant="caption" muted>
        {items.length}/{MAX} photos · La première est votre photo principale. Les photos sont recompressées et nettoyées de leurs
        métadonnées (position GPS comprise).
      </AppText>
      {busy ? <Loading /> : null}
      <ActionSheet
        visible={selected !== null}
        onClose={() => setSelected(null)}
        title="Cette photo"
        actions={[
          ...(index > 0 ? [{ label: "Définir comme photo principale", onPress: () => selected && setMain.mutate(selected.id) }] : []),
          ...(index > 0 ? [{ label: "Déplacer vers la gauche", onPress: () => move(index, -1) }] : []),
          ...(index >= 0 && index < items.length - 1 ? [{ label: "Déplacer vers la droite", onPress: () => move(index, 1) }] : []),
          { label: "Remplacer", onPress: () => selected && replace.mutate(selected.id) },
          { label: "Supprimer", destructive: true, onPress: () => selected && confirmDelete(selected) }
        ]}
      />
    </View>
  );
}
