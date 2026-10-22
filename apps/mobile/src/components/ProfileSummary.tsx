import React from "react";
import { StyleSheet, View } from "react-native";
import type { Interest } from "../api/types";
import { distanceLabel } from "../lib/format";
import { Chip, Text, spacing } from "../shared/ui";

type ProfileSummaryProps = {
  firstName: string;
  age: number;
  city?: string;
  distanceKm?: number;
  bio?: string;
  interests?: Interest[];
  inverse?: boolean;
  compact?: boolean;
};

/** Name, age, place, bio and interests. Used on the swipe card and the full profile. */
export function ProfileSummary({
  firstName,
  age,
  city,
  distanceKm,
  bio,
  interests = [],
  inverse = false,
  compact = false
}: ProfileSummaryProps) {
  const tone = inverse ? "inverse" : "default";
  const place = [city, distanceLabel(distanceKm)].filter(Boolean).join(" · ");
  return (
    <View style={styles.root}>
      <Text accessibilityRole="header" tone={tone} variant="title" weight="bold">
        {firstName}, {age}
      </Text>
      {place ? (
        <Text tone={inverse ? "inverse" : "muted"} variant="label" weight="medium">
          {place}
        </Text>
      ) : null}
      {bio ? (
        <Text numberOfLines={compact ? 2 : undefined} tone={tone}>
          {bio}
        </Text>
      ) : null}
      {interests.length > 0 ? (
        <View style={styles.chips}>
          {interests.slice(0, compact ? 3 : undefined).map((interest) => (
            <Chip key={interest.id} label={interest.label} />
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, marginTop: spacing.xs }
});
