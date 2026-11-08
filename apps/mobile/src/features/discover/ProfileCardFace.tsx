import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { LinearGradient } from "expo-linear-gradient";
import type { ProfileCard } from "../../api/types";
import { formatDistance } from "../../lib/format";
import { radii, spacing } from "../../theme/theme";
import { Photo } from "../../ui/Photo";
import { Text } from "../../ui/Text";

type Props = { card: ProfileCard; onOpenProfile: () => void };

/** Front of a discovery card: photo carousel (tap left/right), identity and a few interests. */
export function ProfileCardFace({ card, onOpenProfile }: Props) {
  const [index, setIndex] = useState(0);
  const photos = card.photos;
  const current = photos[Math.min(index, photos.length - 1)];
  const distance = formatDistance(card.distanceKm);

  return (
    <View style={styles.fill}>
      <Photo
        path={current?.url}
        style={styles.fill}
        accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${card.firstName}`}
      />

      <View style={styles.indicators} pointerEvents="none">
        {photos.map((p, i) => (
          <View key={p.id} style={[styles.bar, { opacity: i === index ? 1 : 0.4 }]} />
        ))}
      </View>

      <View style={styles.tapZones}>
        <Pressable
          style={styles.zone}
          accessibilityLabel="Photo précédente"
          onPress={() => setIndex((i) => Math.max(0, i - 1))}
        />
        <Pressable
          style={styles.zone}
          accessibilityLabel="Photo suivante"
          onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))}
        />
      </View>

      <LinearGradient
        colors={["transparent", "rgba(20,10,8,0.85)"]}
        style={styles.footer}
        pointerEvents="box-none"
      >
        <View style={styles.footerRow} pointerEvents="box-none">
          <View style={styles.flex} pointerEvents="none">
            <Text variant="title" style={styles.white}>
              {card.firstName}, {card.age}
            </Text>
            <View style={styles.meta}>
              <Ionicons name="location-outline" size={14} color="#F5E9E3" />
              <Text variant="label" style={styles.light}>
                {[card.city, distance].filter(Boolean).join(" · ")}
              </Text>
            </View>
            <View style={styles.tags}>
              {card.interests.slice(0, 3).map((i) => (
                <View key={i.slug} style={styles.tag}>
                  <Text variant="caption" style={styles.white}>
                    {i.label}
                  </Text>
                </View>
              ))}
            </View>
          </View>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Voir le profil de ${card.firstName}`}
            onPress={onOpenProfile}
            style={styles.info}
            testID="card-open-profile"
          >
            <Ionicons name="information-circle-outline" size={30} color="#FFFFFF" />
          </Pressable>
        </View>
      </LinearGradient>
    </View>
  );
}

const styles = StyleSheet.create({
  fill: { ...StyleSheet.absoluteFillObject },
  flex: { flex: 1 },
  indicators: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.sm,
    right: spacing.sm,
    flexDirection: "row",
    gap: 4
  },
  bar: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "#FFFFFF" },
  tapZones: { ...StyleSheet.absoluteFillObject, bottom: 160, flexDirection: "row" },
  zone: { flex: 1 },
  footer: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.md,
    paddingTop: spacing.xxl,
    borderBottomLeftRadius: radii.lg,
    borderBottomRightRadius: radii.lg
  },
  footerRow: { flexDirection: "row", alignItems: "flex-end" },
  white: { color: "#FFFFFF" },
  light: { color: "#F5E9E3" },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, marginTop: 2 },
  tags: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: spacing.xs },
  tag: {
    backgroundColor: "rgba(255,255,255,0.2)",
    borderRadius: radii.pill,
    paddingHorizontal: 10,
    paddingVertical: 3
  },
  info: { padding: 4 }
});
