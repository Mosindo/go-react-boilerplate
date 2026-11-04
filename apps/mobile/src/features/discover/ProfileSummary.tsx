import React from "react";
import { View } from "react-native";
import type { PublicProfile } from "../../api/types";
import { Chip, Text, spacing } from "../../shared/ui";

export function distanceLabel(p: Pick<PublicProfile, "distanceKm" | "city">): string {
  const parts: string[] = [];
  if (p.city) parts.push(p.city);
  if (p.distanceKm !== null) parts.push(`à ~${p.distanceKm} km`);
  return parts.join(" · ");
}

/** Bio + interests block shared by the full profile screen. */
export function ProfileBody({ profile }: { profile: PublicProfile }) {
  return (
    <View style={{ gap: spacing.md }}>
      <View style={{ gap: 2 }}>
        <Text variant="title">{profile.firstName}, {profile.age}</Text>
        {distanceLabel(profile) ? <Text tone="muted">{distanceLabel(profile)}</Text> : null}
      </View>
      {profile.bio ? <Text>{profile.bio}</Text> : null}
      {profile.interests.length > 0 ? (
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {profile.interests.map((i) => <Chip key={i.slug} label={i.label} />)}
        </View>
      ) : null}
    </View>
  );
}
