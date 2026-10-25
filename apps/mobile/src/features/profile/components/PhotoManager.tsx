import React, { useState } from "react";
import { Alert, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as ImagePicker from "expo-image-picker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { photosApi, queryKeys, type Photo, type ProfileState } from "../../../api";
import { MAX_PHOTOS } from "../../../config";
import { showToast } from "../../../shared/feedback";
import { ActionSheet, AuthImage, Text, type ActionSheetOption } from "../../../shared/ui";
import { radius, spacing, useTheme } from "../../../theme";
import { addPhoto, movePhotoId, removePhoto, replacePhoto, setPhotoOrder } from "../photoCache";
import { toUploadFile } from "../photoFile";

const COLUMNS = 3;
const GAP = spacing.md;

async function pickImage() {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    showToast("Autorisez l'accès à vos photos dans les réglages pour continuer.", "error");
    return null;
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 0.85
  });
  const asset = result.canceled ? undefined : result.assets[0];
  return asset ? toUploadFile(asset) : null;
}

type Props = { photos: Photo[]; allowDeleteLast?: boolean };

export function PhotoManager({ photos, allowDeleteLast = false }: Props) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<Photo | null>(null);
  const slotWidth = Math.floor((width - spacing.xl * 2 - GAP * (COLUMNS - 1)) / COLUMNS);

  const updateProfile = (update: (state: ProfileState | undefined) => ProfileState | undefined) => {
    queryClient.setQueryData<ProfileState>(queryKeys.profile, update);
  };

  const upload = useMutation({
    mutationFn: photosApi.upload,
    onSuccess: (photo) => updateProfile((state) => addPhoto(state, photo))
  });
  const replace = useMutation({
    mutationFn: ({ id, file }: { id: string; file: Parameters<typeof photosApi.upload>[0] }) =>
      photosApi.replace(id, file),
    onSuccess: (photo, { id }) => updateProfile((state) => replacePhoto(state, id, photo))
  });
  const reorder = useMutation({
    mutationFn: photosApi.reorder,
    onSuccess: (ordered) => updateProfile((state) => setPhotoOrder(state, ordered))
  });
  const remove = useMutation({
    mutationFn: photosApi.remove,
    onSuccess: (_void, photoId) => updateProfile((state) => removePhoto(state, photoId))
  });
  const busy = upload.isPending || replace.isPending || reorder.isPending || remove.isPending;

  const addNew = async () => {
    const file = await pickImage();
    if (file) upload.mutate(file);
  };

  const replaceSelected = async (photo: Photo) => {
    const file = await pickImage();
    if (file) replace.mutate({ id: photo.id, file });
  };

  const move = (photo: Photo, offset: number) => {
    const ids = movePhotoId(photos, photo.id, offset);
    if (ids) reorder.mutate(ids);
  };

  const confirmRemove = (photo: Photo) => {
    if (photos.length <= 1 && !allowDeleteLast) {
      showToast("Au moins une photo est requise.", "error");
      return;
    }
    Alert.alert("Supprimer cette photo ?", "Cette action est définitive.", [
      { text: "Annuler", style: "cancel" },
      { text: "Supprimer", style: "destructive", onPress: () => remove.mutate(photo.id) }
    ]);
  };

  const options: ActionSheetOption[] = selected
    ? [
        { label: "Remplacer", onPress: () => void replaceSelected(selected), testID: "photo-replace" },
        ...(selected.position > 0
          ? [
              { label: "Définir comme photo principale", onPress: () => move(selected, -selected.position) },
              { label: "Déplacer vers la gauche", onPress: () => move(selected, -1) }
            ]
          : []),
        ...(selected.position < photos.length - 1
          ? [{ label: "Déplacer vers la droite", onPress: () => move(selected, 1) }]
          : []),
        { label: "Supprimer", destructive: true, onPress: () => confirmRemove(selected), testID: "photo-delete" }
      ]
    : [];

  return (
    <View style={styles.grid}>
      {Array.from({ length: MAX_PHOTOS }, (_unused, index) => {
        const photo = photos[index];
        const slotStyle = { width: slotWidth, height: Math.round(slotWidth * 1.33), borderColor: colors.border };
        if (photo) {
          return (
            <Pressable
              accessibilityHint="Ouvre les actions pour cette photo"
              accessibilityLabel={`Photo ${index + 1}${photo.isPrimary ? ", principale" : ""}`}
              accessibilityRole="button"
              key={photo.id}
              onPress={() => setSelected(photo)}
              style={[styles.slot, slotStyle]}
              testID={`photo-slot-${index}`}
            >
              <AuthImage path={photo.thumbUrl} style={styles.fill} />
              {photo.isPrimary ? (
                <View style={[styles.tag, { backgroundColor: colors.scrim }]}>
                  <Text style={{ color: colors.onScrim }} variant="caption">
                    Principale
                  </Text>
                </View>
              ) : null}
            </Pressable>
          );
        }
        const isNextFree = index === photos.length;
        return (
          <Pressable
            accessibilityLabel="Ajouter une photo"
            accessibilityRole="button"
            accessibilityState={{ disabled: !isNextFree || busy }}
            disabled={!isNextFree || busy}
            key={`empty-${index}`}
            onPress={() => void addNew()}
            style={[
              styles.slot,
              styles.empty,
              slotStyle,
              { backgroundColor: colors.surface, opacity: isNextFree ? 1 : 0.5 }
            ]}
            testID={isNextFree ? "photo-add-button" : undefined}
          >
            <Ionicons
              color={colors.primary}
              name={isNextFree && upload.isPending ? "hourglass-outline" : "add"}
              size={32}
            />
          </Pressable>
        );
      })}
      <ActionSheet onClose={() => setSelected(null)} options={options} title="Photo" visible={selected !== null} />
    </View>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: GAP },
  slot: { borderRadius: radius.md, overflow: "hidden", borderWidth: 1 },
  empty: { alignItems: "center", justifyContent: "center", borderStyle: "dashed" },
  fill: { width: "100%", height: "100%" },
  tag: {
    position: "absolute",
    left: spacing.xs,
    bottom: spacing.xs,
    borderRadius: radius.sm,
    paddingHorizontal: 6,
    paddingVertical: 2
  }
});
