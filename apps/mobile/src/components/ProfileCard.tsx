import React, { useCallback, useEffect, useState } from "react";
import { Image, Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/models";
import { formatNameAge, formatPlace, humanizeSlug } from "../lib/dating/format";
import { useTheme } from "../shared/ui/theme";
import { Chip, IconButton, Txt, photoTextColor } from "./kit";
import { photoUri } from "./photoUrl";
import { Scrim } from "./Scrim";

type Props = {
  profile: PublicProfile;
  /** Opens the full profile view. Omit for non-interactive (background) cards. */
  onOpenInfo?: (profile: PublicProfile) => void;
  interactive?: boolean;
  testID?: string;
};

const MAX_CHIPS = 4;

function ProfileCardBase({ profile, onOpenInfo, interactive = true, testID }: Props) {
  const { colors, radii, spacing } = useTheme();
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const count = photos.length;
  const current = photos[Math.min(index, Math.max(0, count - 1))];
  const uri = photoUri(current);

  useEffect(() => {
    const next = photos[index + 1];
    const nextUri = photoUri(next);
    if (nextUri) void Image.prefetch(nextUri).catch(() => undefined);
  }, [photos, index]);

  const goPrev = useCallback(() => setIndex((i) => Math.max(0, i - 1)), []);
  const goNext = useCallback(() => setIndex((i) => Math.min(count - 1, i + 1)), [count]);
  const openInfo = useCallback(() => onOpenInfo?.(profile), [onOpenInfo, profile]);

  const title = formatNameAge(profile.firstName, profile.age);
  const place = formatPlace(profile.city, profile.distanceKm);
  const chips = profile.interests.slice(0, MAX_CHIPS);

  return (
    <View
      testID={testID}
      style={[
        styles.card,
        {
          borderRadius: radii.xl,
          backgroundColor: colors.surfaceMuted,
          borderColor: colors.border
        }
      ]}
    >
      {uri ? (
        <Image
          source={{ uri }}
          resizeMode="cover"
          style={StyleSheet.absoluteFill}
          accessibilityIgnoresInvertColors
          accessible={false}
        />
      ) : (
        <View style={[StyleSheet.absoluteFill, styles.noPhoto]}>
          <Txt variant="title" tone="muted">
            {profile.firstName.charAt(0).toUpperCase()}
          </Txt>
        </View>
      )}

      {interactive && count > 1 ? (
        <>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Previous photo"
            onPress={goPrev}
            style={[styles.zone, styles.zoneLeft]}
            testID={testID ? `${testID}-prev-photo` : undefined}
          />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Next photo"
            onPress={goNext}
            style={[styles.zone, styles.zoneRight]}
            testID={testID ? `${testID}-next-photo` : undefined}
          />
        </>
      ) : null}

      {count > 1 ? (
        <View
          pointerEvents="none"
          accessible
          accessibilityLabel={`Photo ${Math.min(index, count - 1) + 1} of ${count}`}
          style={[styles.dots, { top: spacing.sm, left: spacing.md, right: spacing.md }]}
        >
          {photos.map((p, i) => (
            <View
              key={p.id}
              style={[
                styles.dot,
                {
                  backgroundColor: photoTextColor,
                  opacity: i === Math.min(index, count - 1) ? 0.95 : 0.35
                }
              ]}
            />
          ))}
        </View>
      ) : null}

      <Scrim height={230} />

      <View
        pointerEvents="box-none"
        style={[styles.info, { padding: spacing.lg, gap: spacing.sm }]}
      >
        <Pressable
          disabled={!interactive || !onOpenInfo}
          onPress={openInfo}
          accessibilityRole="button"
          accessibilityLabel={`${title}${place ? `. ${place}` : ""}. Open full profile`}
          testID={testID ? `${testID}-info` : undefined}
          style={[styles.flex, { gap: spacing.xs }]}
        >
          <Txt variant="title" tone="inverse" numberOfLines={1}>
            {title}
          </Txt>
          {place ? (
            <Txt tone="inverse" numberOfLines={1}>
              {place}
            </Txt>
          ) : null}
          {chips.length > 0 ? (
            <View style={[styles.chips, { gap: spacing.xs }]}>
              {chips.map((slug) => (
                <Chip key={slug} label={humanizeSlug(slug)} onPhoto />
              ))}
            </View>
          ) : null}
        </Pressable>
        {interactive && onOpenInfo ? (
          <IconButton
            glyph="i"
            label={`Open ${profile.firstName}'s profile`}
            onPress={openInfo}
            color={photoTextColor}
            borderColor={photoTextColor}
            glyphSize={18}
          />
        ) : null}
      </View>
    </View>
  );
}

export const ProfileCard = React.memo(ProfileCardBase);

const styles = StyleSheet.create({
  card: { flex: 1, overflow: "hidden", borderWidth: StyleSheet.hairlineWidth },
  noPhoto: { alignItems: "center", justifyContent: "center" },
  zone: { position: "absolute", top: 0, bottom: 200, width: "40%" },
  zoneLeft: { left: 0 },
  zoneRight: { right: 0 },
  dots: { position: "absolute", flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    alignItems: "flex-end"
  },
  flex: { flex: 1 },
  chips: { flexDirection: "row", flexWrap: "wrap", marginTop: 4 }
});
