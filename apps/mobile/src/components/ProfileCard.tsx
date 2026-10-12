import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Card } from "../api/types";
import { formatDistance } from "../lib/format";
import { radii, spacing } from "../theme";
import { Chip, Text } from "../ui";
import { PhotoCarousel } from "./PhotoCarousel";

type Props = { card: Card; onOpenDetails?: () => void };

/** Photo-first card used in the discovery deck. */
export function ProfileCard({ card, onOpenDetails }: Props) {
  const distance = formatDistance(card.distanceKm);
  const place = [card.city, distance].filter(Boolean).join(" · ");
  return (
    <View style={styles.card} accessible accessibilityLabel={`${card.firstName}, ${card.age} ans`}>
      <PhotoCarousel photos={card.photos} name={card.firstName} />
      <View style={styles.scrim} pointerEvents="box-none">
        <View style={styles.row}>
          <View style={styles.flex}>
            <Text variant="title" style={styles.white}>
              {card.firstName}, {card.age}
            </Text>
            {place ? (
              <Text variant="label" style={styles.whiteSoft}>
                {place}
              </Text>
            ) : null}
          </View>
          {onOpenDetails ? (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Voir le profil de ${card.firstName}`}
              onPress={onOpenDetails}
              style={styles.info}
              hitSlop={8}
            >
              <Ionicons name="information" size={22} color="#FFFFFF" />
            </Pressable>
          ) : null}
        </View>
        {card.interests.length > 0 ? (
          <View style={styles.tags} pointerEvents="none">
            {card.interests.slice(0, 3).map((i) => (
              <View key={i.slug} style={styles.tag}>
                <Text variant="caption" style={styles.white}>
                  {i.label}
                </Text>
              </View>
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

/** Full read-only presentation of a profile. */
export function ProfileDetails({ card }: { card: Card }) {
  const distance = formatDistance(card.distanceKm);
  return (
    <View style={styles.details}>
      <View>
        <Text variant="title">
          {card.firstName}, {card.age}
        </Text>
        {card.occupation ? <Text tone="muted">{card.occupation}</Text> : null}
        {[card.city, distance].filter(Boolean).length > 0 ? (
          <Text tone="muted">{[card.city, distance].filter(Boolean).join(" · ")}</Text>
        ) : null}
      </View>
      {card.bio ? <Text>{card.bio}</Text> : null}
      {card.interests.length > 0 ? (
        <View style={styles.chips}>
          {card.interests.map((i) => (
            <Chip key={i.slug} label={i.label} />
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.xl, overflow: "hidden", backgroundColor: "#000" },
  flex: { flex: 1 },
  scrim: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxl,
    gap: spacing.sm,
    backgroundColor: "rgba(0,0,0,0.38)"
  },
  row: { flexDirection: "row", alignItems: "flex-end", gap: spacing.md },
  white: { color: "#FFFFFF" },
  whiteSoft: { color: "rgba(255,255,255,0.88)" },
  info: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: "rgba(255,255,255,0.25)",
    alignItems: "center",
    justifyContent: "center"
  },
  tags: { flexDirection: "row", gap: spacing.sm, flexWrap: "wrap" },
  tag: { backgroundColor: "rgba(255,255,255,0.22)", borderRadius: radii.pill, paddingHorizontal: spacing.md, paddingVertical: 4 },
  details: { gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
