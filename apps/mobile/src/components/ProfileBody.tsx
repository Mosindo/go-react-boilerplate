import React from "react";
import { View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Gender, Interest, PhotoView } from "../api/types";
import { genderShort } from "../i18n/fr";
import { useTheme } from "../theme";
import { Chip, Text, spacing } from "../shared/ui";
import { PhotoCarousel } from "./PhotoCarousel";

type ProfileBodyProps = {
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city?: string;
  distanceKm?: number;
  interests: Interest[];
  photos: PhotoView[];
  /** Slugs the viewer shares; highlighted so common ground is visible at a glance. */
  highlightSlugs?: ReadonlySet<string>;
};

export function distanceLabel(distanceKm: number | undefined): string | null {
  return distanceKm === undefined ? null : `À moins de ${distanceKm} km`;
}

/** The full profile layout shared by "someone else's profile" and "my profile preview". */
export function ProfileBody({ age, bio, city, distanceKm, firstName, gender, highlightSlugs, interests, photos }: ProfileBodyProps) {
  const palette = useTheme();
  const distance = distanceLabel(distanceKm);

  return (
    <View style={{ gap: spacing.lg }}>
      <View style={{ aspectRatio: 3 / 4, borderRadius: 28, overflow: "hidden" }}>
        <PhotoCarousel name={firstName} photos={photos} />
      </View>

      <View style={{ gap: spacing.xs }}>
        <Text accessibilityRole="header" variant="title" weight="bold">
          {firstName}, {age}
        </Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", alignItems: "center", gap: spacing.md }}>
          <Text tone="muted">{genderShort[gender]}</Text>
          {city ? (
            <View style={{ flexDirection: "row", alignItems: "center", gap: 4 }}>
              <Ionicons color={palette.textMuted} name="location-outline" size={16} />
              <Text tone="muted">{city}</Text>
            </View>
          ) : null}
          {distance ? <Text tone="muted">{distance}</Text> : null}
        </View>
      </View>

      {bio ? (
        <View style={{ gap: spacing.xs }}>
          <Text tone="muted" variant="eyebrow" weight="semibold">
            À propos
          </Text>
          <Text selectable>{bio}</Text>
        </View>
      ) : null}

      {interests.length > 0 ? (
        <View style={{ gap: spacing.sm }}>
          <Text tone="muted" variant="eyebrow" weight="semibold">
            Centres d&apos;intérêt
          </Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
            {interests.map((interest) => (
              <Chip key={interest.slug} label={interest.label} selected={highlightSlugs?.has(interest.slug) ?? false} />
            ))}
          </View>
        </View>
      ) : null}
    </View>
  );
}
