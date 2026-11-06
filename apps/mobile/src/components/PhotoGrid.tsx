import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/errors";
import { photoApi } from "../api/services";
import type { Photo } from "../api/types";
import { fr } from "../i18n/fr";
import { pickPhotoForm } from "../lib/device";
import { radius, spacing, useTheme } from "../theme";
import { AuthImage } from "./AuthImage";
import { useToast } from "./Toast";
import { Button, Text } from "./ui";

/** Add / replace / delete / reorder up to 6 photos. Position 0 is the main photo. */
export function PhotoManager({ photos, onChanged }: { photos: Photo[]; onChanged: () => void }) {
  const { colors } = useTheme();
  const toast = useToast();
  const [busy, setBusy] = useState<string | null>(null);

  const run = async (key: string, action: () => Promise<unknown>) => {
    setBusy(key);
    try {
      await action();
      onChanged();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(null);
    }
  };

  const add = async () => {
    const form = await pickPhotoForm();
    if (form === "denied") return toast.show(fr.profile.permissionPhotos, "error");
    if (!form) return;
    await run("add", () => photoApi.upload(form));
  };

  const replace = async (id: string) => {
    const form = await pickPhotoForm();
    if (form === "denied") return toast.show(fr.profile.permissionPhotos, "error");
    if (!form) return;
    await run(id, () => photoApi.replace(id, form));
  };

  const move = (index: number, to: number) => {
    const ids = photos.map((p) => p.id);
    const [moved] = ids.splice(index, 1);
    if (!moved) return;
    ids.splice(to, 0, moved);
    void run("order", () => photoApi.reorder(ids));
  };

  const slots = Array.from({ length: 6 }, (_, i) => photos[i] ?? null);
  return (
    <View style={{ gap: spacing.md }}>
      <View style={styles.grid}>
        {slots.map((photo, i) => (
          <View key={photo?.id ?? `empty-${i}`} style={styles.cell}>
            {photo ? (
              <View
                style={[styles.slot, { borderColor: colors.border }]}
                testID={`photo-slot-${i}`}
              >
                <AuthImage
                  photoId={photo.id}
                  style={StyleSheet.absoluteFill as object}
                  accessibilityLabel={`Photo ${i + 1}`}
                />
                {i === 0 ? (
                  <View style={[styles.mainBadge, { backgroundColor: colors.primary }]}>
                    <Text
                      variant="caption"
                      color={colors.primaryText}
                      style={{ fontWeight: "700" }}
                    >
                      {fr.profile.mainBadge}
                    </Text>
                  </View>
                ) : null}
                <View style={[styles.tools, { backgroundColor: colors.scrim }]}>
                  {i > 0 ? (
                    <Tool label={fr.profile.moveLeft} glyph="←" onPress={() => move(i, i - 1)} />
                  ) : null}
                  {i < photos.length - 1 ? (
                    <Tool label={fr.profile.moveRight} glyph="→" onPress={() => move(i, i + 1)} />
                  ) : null}
                  {i > 0 ? (
                    <Tool label={fr.profile.makeMain} glyph="★" onPress={() => move(i, 0)} />
                  ) : null}
                  <Tool
                    label={fr.profile.replace}
                    glyph="↻"
                    onPress={() => void replace(photo.id)}
                  />
                  <Tool
                    testID={`photo-delete-${i}`}
                    label={fr.common.delete}
                    glyph="✕"
                    onPress={() => void run(photo.id, () => photoApi.remove(photo.id))}
                  />
                </View>
                {busy === photo.id ? (
                  <View style={[StyleSheet.absoluteFill, { backgroundColor: colors.overlay }]} />
                ) : null}
              </View>
            ) : (
              <Pressable
                testID={i === photos.length ? "photo-add" : undefined}
                accessibilityRole="button"
                accessibilityLabel={fr.profile.addPhoto}
                disabled={i !== photos.length || busy !== null}
                onPress={() => void add()}
                style={[
                  styles.slot,
                  styles.empty,
                  { borderColor: colors.border, backgroundColor: colors.surfaceAlt }
                ]}
              >
                <Text variant="title" muted>
                  {i === photos.length ? "+" : ""}
                </Text>
              </Pressable>
            )}
          </View>
        ))}
      </View>
      {busy === "add" ? (
        <Button title={fr.common.loading} loading onPress={() => undefined} />
      ) : null}
    </View>
  );
}

function Tool({
  label,
  glyph,
  onPress,
  testID
}: {
  label: string;
  glyph: string;
  onPress: () => void;
  testID?: string;
}) {
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      hitSlop={4}
      style={styles.tool}
    >
      <Text color="#FFFFFF" style={{ fontSize: 15, fontWeight: "700" }}>
        {glyph}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", marginHorizontal: -spacing.xs },
  cell: { width: "33.33%", padding: spacing.xs },
  slot: { aspectRatio: 3 / 4, borderRadius: radius.md, borderWidth: 1, overflow: "hidden" },
  empty: { alignItems: "center", justifyContent: "center", borderStyle: "dashed" },
  mainBadge: {
    position: "absolute",
    top: 6,
    left: 6,
    borderRadius: radius.pill,
    paddingHorizontal: 8,
    paddingVertical: 2
  },
  tools: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    flexWrap: "wrap",
    justifyContent: "center"
  },
  tool: { minWidth: 30, minHeight: 30, alignItems: "center", justifyContent: "center" }
});
