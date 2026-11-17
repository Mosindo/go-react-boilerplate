import React, { useMemo, useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage, mediaUrl } from "../../api/client";
import { photosApi, profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Gender, Interest, MyProfile, Photo } from "../../api/types";
import { colors, radii, spacing } from "../../theme";
import { Button, Chip, Text, TextField, showToast } from "../../ui";
import { CITIES, getApproximatePosition, pickPhoto } from "../../utils/device";
import { GENDER_LABELS, INTERESTED_LABELS, maskFrenchDate, type GenderValue } from "../../utils/format";

const GENDERS: GenderValue[] = ["woman", "man", "nonbinary"];
export const MAX_INTERESTS = 10;
export const MAX_PHOTOS = 6;

export function BirthdateField({ value, onChange, error }: { value: string; onChange: (v: string) => void; error?: string | null }) {
  return (
    <TextField
      label="Date de naissance"
      testID="field-birthdate"
      placeholder="JJ/MM/AAAA"
      keyboardType="number-pad"
      value={value}
      onChangeText={(v) => onChange(maskFrenchDate(v))}
      maxLength={10}
      error={error}
      hint="Seul ton âge sera visible."
    />
  );
}

export function GenderPicker({ value, onChange }: { value: Gender | null; onChange: (g: Gender) => void }) {
  return (
    <View style={styles.wrap} accessibilityRole="radiogroup">
      {GENDERS.map((g) => (
        <Chip key={g} testID={`gender-${g}`} label={GENDER_LABELS[g]} selected={value === g} onPress={() => onChange(g)} />
      ))}
    </View>
  );
}

export function InterestedInPicker({ value, onChange }: { value: Gender[]; onChange: (g: Gender[]) => void }) {
  return (
    <View style={styles.wrap}>
      {GENDERS.map((g) => {
        const selected = value.includes(g);
        return (
          <Chip
            key={g}
            testID={`interested-${g}`}
            label={INTERESTED_LABELS[g]}
            selected={selected}
            onPress={() => {
              if (selected && value.length === 1) return; // keep at least one
              onChange(selected ? value.filter((x) => x !== g) : [...value, g]);
            }}
          />
        );
      })}
    </View>
  );
}

const CATEGORY_LABELS: Record<string, string> = {
  outdoors: "Plein air",
  sport: "Sport",
  food: "Table",
  arts: "Culture",
  music: "Musique",
  games: "Jeux",
  mind: "Curiosité",
  lifestyle: "Art de vivre"
};

export function InterestsPicker({ all, selected, onChange }: { all: Interest[]; selected: number[]; onChange: (ids: number[]) => void }) {
  const groups = useMemo(() => {
    const map = new Map<string, Interest[]>();
    all.forEach((i) => map.set(i.category, [...(map.get(i.category) ?? []), i]));
    return Array.from(map.entries());
  }, [all]);
  const full = selected.length >= MAX_INTERESTS;
  return (
    <View style={styles.groups}>
      <Text variant="caption" tone="subtle">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      {groups.map(([category, items]) => (
        <View key={category} style={styles.group}>
          <Text variant="overline" tone="subtle">
            {CATEGORY_LABELS[category] ?? category}
          </Text>
          <View style={styles.wrap}>
            {items.map((i) => {
              const isSelected = selected.includes(i.id);
              return (
                <Chip
                  key={i.id}
                  testID={`interest-${i.slug}`}
                  label={i.label}
                  selected={isSelected}
                  disabled={full && !isSelected}
                  onPress={() => onChange(isSelected ? selected.filter((x) => x !== i.id) : [...selected, i.id])}
                />
              );
            })}
          </View>
        </View>
      ))}
    </View>
  );
}

/** Six-slot photo manager: add, replace, delete and choose the main photo. */
export function PhotoGrid({ photos }: { photos: Photo[] }) {
  const queryClient = useQueryClient();
  const [busySlot, setBusySlot] = useState<string | null>(null);

  const apply = (next: Photo[]) => {
    queryClient.setQueryData<MyProfile>(queryKeys.profile, (p) => (p ? { ...p, photos: next } : p));
    void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
  };

  const mutation = useMutation({
    mutationFn: async (action: { kind: "add" } | { kind: "replace" | "delete" | "primary"; photo: Photo }) => {
      if (action.kind === "delete") return (await photosApi.remove(action.photo.id)).photos;
      if (action.kind === "primary") {
        const order = [action.photo.id, ...photos.filter((p) => p.id !== action.photo.id).map((p) => p.id)];
        return (await photosApi.reorder(order)).photos;
      }
      const picked = await pickPhoto();
      if (!picked) return null;
      if (action.kind === "replace") return (await photosApi.replace(action.photo.id, picked)).photos;
      return (await photosApi.upload(picked)).photos;
    },
    onMutate: (action) => setBusySlot(action.kind === "add" ? "add" : action.photo.id),
    onSuccess: (next) => {
      if (next) apply(next);
    },
    onError: (error) => showToast(errorMessage(error), "error"),
    onSettled: () => setBusySlot(null)
  });

  const slots = Array.from({ length: MAX_PHOTOS }, (_, i) => photos[i] ?? null);
  return (
    <View style={styles.grid} testID="photo-grid">
      {slots.map((photo, index) => {
        if (!photo) {
          const isNext = index === photos.length;
          return (
            <Pressable
              key={`empty-${index}`}
              testID={isNext ? "photo-add" : undefined}
              accessibilityRole="button"
              accessibilityLabel="Ajouter une photo"
              disabled={!isNext || mutation.isPending}
              onPress={() => mutation.mutate({ kind: "add" })}
              style={[styles.slot, styles.emptySlot, isNext ? styles.nextSlot : null]}
            >
              {busySlot === "add" && isNext ? (
                <ActivityIndicator color={colors.accent} />
              ) : (
                <Ionicons name="add" size={28} color={isNext ? colors.accent : colors.borderStrong} />
              )}
            </Pressable>
          );
        }
        return (
          <View key={photo.id} style={styles.slot}>
            <Image
              source={{ uri: mediaUrl(photo.url) }}
              style={StyleSheet.absoluteFill}
              contentFit="cover"
              accessibilityLabel={`Photo ${index + 1}`}
            />
            {busySlot === photo.id ? (
              <View style={styles.busy}>
                <ActivityIndicator color={colors.white} />
              </View>
            ) : null}
            {index === 0 ? (
              <View style={styles.mainBadge}>
                <Text variant="caption" tone="inverse">
                  Principale
                </Text>
              </View>
            ) : null}
            <View style={styles.slotActions}>
              {index > 0 ? (
                <SlotAction
                  icon="star-outline"
                  label="Définir comme photo principale"
                  onPress={() => mutation.mutate({ kind: "primary", photo })}
                />
              ) : null}
              <SlotAction icon="swap-horizontal" label="Remplacer la photo" onPress={() => mutation.mutate({ kind: "replace", photo })} />
              <SlotAction icon="trash-outline" label="Supprimer la photo" onPress={() => mutation.mutate({ kind: "delete", photo })} />
            </View>
          </View>
        );
      })}
    </View>
  );
}

function SlotAction({ icon, label, onPress }: { icon: React.ComponentProps<typeof Ionicons>["name"]; label: string; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={label} hitSlop={6} onPress={onPress} style={styles.slotAction}>
      <Ionicons name={icon} size={16} color={colors.white} />
    </Pressable>
  );
}

/** Lets the user share an approximate position, or choose a city instead of GPS. */
export function LocationPicker({
  currentCity,
  hasLocation,
  successMessage = "Position mise à jour."
}: {
  currentCity: string;
  hasLocation: boolean;
  successMessage?: string;
}) {
  const queryClient = useQueryClient();
  const [showCities, setShowCities] = useState(false);
  const save = useMutation({
    mutationFn: async (source: { kind: "gps" } | { kind: "city"; latitude: number; longitude: number; city: string }) => {
      if (source.kind === "gps") {
        const pos = await getApproximatePosition();
        return profileApi.updateLocation(pos.latitude, pos.longitude, pos.city);
      }
      return profileApi.updateLocation(source.latitude, source.longitude, source.city);
    },
    onSuccess: (profile) => {
      queryClient.setQueryData(queryKeys.profile, profile);
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      showToast(successMessage, "success");
      setShowCities(false);
    },
    onError: (error) => {
      showToast(errorMessage(error), "error");
      setShowCities(true);
    }
  });

  return (
    <View style={styles.location}>
      {hasLocation ? (
        <View style={styles.currentLocation}>
          <Ionicons name="location" size={18} color={colors.accent} />
          <Text>{currentCity || "Position enregistrée"}</Text>
        </View>
      ) : null}
      <Button
        label="Utiliser ma position approximative"
        testID="location-gps"
        icon={<Ionicons name="navigate" size={18} color={colors.accentText} />}
        loading={save.isPending && save.variables?.kind === "gps"}
        onPress={() => save.mutate({ kind: "gps" })}
      />
      <Button
        label={showCities ? "Masquer les villes" : "Choisir une ville"}
        variant="secondary"
        testID="location-city-toggle"
        onPress={() => setShowCities((v) => !v)}
      />
      {showCities ? (
        <View style={styles.wrap}>
          {CITIES.map((c) => (
            <Chip
              key={c.city}
              testID={`city-${c.city}`}
              label={c.city ?? ""}
              selected={currentCity === c.city}
              onPress={() => save.mutate({ kind: "city", latitude: c.latitude, longitude: c.longitude, city: c.city ?? "" })}
            />
          ))}
        </View>
      ) : null}
      <Text variant="caption" tone="subtle">
        Ta position est arrondie à environ 1 km et n’est jamais montrée : les autres voient seulement une distance approximative.
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  },
  groups: {
    gap: spacing.lg
  },
  group: {
    gap: spacing.sm
  },
  grid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  },
  slot: {
    width: "31.5%",
    aspectRatio: 4 / 5,
    borderRadius: radii.md,
    overflow: "hidden",
    backgroundColor: colors.surface
  },
  emptySlot: {
    borderWidth: 1,
    borderStyle: "dashed",
    borderColor: colors.border,
    alignItems: "center",
    justifyContent: "center"
  },
  nextSlot: {
    borderColor: colors.accent,
    backgroundColor: colors.accentSoft
  },
  busy: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: colors.scrim,
    alignItems: "center",
    justifyContent: "center"
  },
  mainBadge: {
    position: "absolute",
    left: spacing.xs,
    top: spacing.xs,
    backgroundColor: colors.gold,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2
  },
  slotActions: {
    position: "absolute",
    right: spacing.xs,
    bottom: spacing.xs,
    flexDirection: "row",
    gap: spacing.xs
  },
  slotAction: {
    width: 28,
    height: 28,
    borderRadius: 14,
    backgroundColor: colors.overlay,
    alignItems: "center",
    justifyContent: "center"
  },
  location: {
    gap: spacing.md
  },
  currentLocation: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  }
});
