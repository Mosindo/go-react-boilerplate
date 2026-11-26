import React from "react";
import { StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { radii, spacing } from "../theme";
import { Icon, Text } from "../ui";
import { formatDistance } from "../utils/format";

/** Compact facts shown over a photo: job, city/distance and the demo marker. */
export function ProfileFacts({ profile, light }: { profile: PublicProfile; light?: boolean }) {
  const tone = light ? "inverse" : "muted";
  const iconColor = light ? "rgba(255,255,255,0.9)" : "#8A7C83";
  const distance = formatDistance(profile.distanceKm);
  const place = [profile.city, distance].filter(Boolean).join(" · ");
  return (
    <View style={styles.facts}>
      {profile.jobTitle ? (
        <View style={styles.fact}>
          <Icon color={iconColor} name="briefcase-outline" size={16} />
          <Text numberOfLines={1} tone={tone} variant="label">
            {profile.jobTitle}
          </Text>
        </View>
      ) : null}
      {place ? (
        <View style={styles.fact}>
          <Icon color={iconColor} name="location-outline" size={16} />
          <Text numberOfLines={1} tone={tone} variant="label">
            {place}
          </Text>
        </View>
      ) : null}
    </View>
  );
}

export function DemoBadge() {
  return (
    <View accessibilityLabel="Profil de démonstration" style={styles.demo}>
      <Text style={styles.demoText} variant="overline">
        Démo
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  facts: { gap: 4 },
  fact: { flexDirection: "row", alignItems: "center", gap: 6 },
  demo: { backgroundColor: "rgba(255,255,255,0.92)", borderRadius: radii.pill, paddingHorizontal: spacing.xs, paddingVertical: 2 },
  demoText: { color: "#5A3A66", fontSize: 10, lineHeight: 14 }
});
