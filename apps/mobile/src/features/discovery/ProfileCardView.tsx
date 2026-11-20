import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { LinearGradient } from "expo-linear-gradient";
import Ionicons from "@expo/vector-icons/Ionicons";
import { mediaUrl } from "../../api/client";
import type { ProfileCard } from "../../api/types";
import { formatDistance } from "../../lib/format";
import { useTheme } from "../../theme";
import { Badge, Text } from "../../ui";

type Props = { card: ProfileCard; onOpen?: () => void };

/** Full-bleed card: tap the left/right half to browse photos. */
export function ProfileCardView({ card, onOpen }: Props) {
  const { colors, radii, spacing } = useTheme();
  const [index, setIndex] = useState(0);
  const photo = card.photos[Math.min(index, card.photos.length - 1)];
  const shared = card.interests.filter((i) => i.shared);
  const highlights = (shared.length ? shared : card.interests).slice(0, 3);
  const distance = formatDistance(card.distanceKm);

  return (
    <View style={[styles.card, { borderRadius: radii.xl, backgroundColor: colors.surfaceAlt }]}>
      {photo ? (
        <Image
          accessibilityLabel={`Photo ${index + 1} de ${card.firstName}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          source={{ uri: mediaUrl(photo.url) }}
          style={StyleSheet.absoluteFill}
          transition={200}
        />
      ) : null}
      {card.photos.length > 1 ? (
        <View style={[styles.indicators, { padding: spacing.sm }]}>
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.indicator, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.35)" }]} />
          ))}
        </View>
      ) : null}
      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" onPress={() => setIndex((i) => Math.max(0, i - 1))} style={styles.flex} />
        <Pressable accessibilityLabel="Photo suivante" onPress={() => setIndex((i) => Math.min(card.photos.length - 1, i + 1))} style={styles.flex} />
      </View>
      <LinearGradient
        colors={["transparent", "rgba(12,8,14,0.55)", "rgba(12,8,14,0.92)"]}
        pointerEvents="box-none"
        style={[styles.info, { padding: spacing.lg, gap: spacing.xs }]}
      >
        {card.isDemo ? <Badge label="Profil démo" tone="accent" /> : null}
        <Pressable accessibilityHint="Voir le profil complet" accessibilityRole="button" onPress={onOpen} style={styles.nameRow}>
          <Text style={styles.white} variant="display">
            {card.firstName}
            <Text style={styles.whiteSoft} variant="title">
              {"  "}
              {card.age}
            </Text>
          </Text>
          <Ionicons color="#FFFFFF" name="information-circle-outline" size={26} />
        </Pressable>
        {card.jobTitle || distance ? (
          <Text style={styles.whiteSoft} variant="bodyStrong">
            {[card.jobTitle, distance].filter(Boolean).join(" · ")}
          </Text>
        ) : null}
        {highlights.length ? (
          <View style={styles.chips}>
            {highlights.map((i) => (
              <View key={i.id} style={[styles.chip, { borderRadius: radii.pill, backgroundColor: i.shared ? colors.primary : "rgba(255,255,255,0.18)" }]}>
                <Text style={{ color: i.shared ? colors.onPrimary : "#FFFFFF" }} variant="caption">
                  {i.shared ? "♥ " : ""}
                  {i.label}
                </Text>
              </View>
            ))}
          </View>
        ) : null}
      </LinearGradient>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, overflow: "hidden" },
  flex: { flex: 1 },
  indicators: { position: "absolute", top: 0, left: 0, right: 0, flexDirection: "row", gap: 4, zIndex: 2 },
  indicator: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, paddingTop: 80 },
  nameRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  white: { color: "#FFFFFF" },
  whiteSoft: { color: "rgba(255,255,255,0.85)" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: 4 },
  chip: { paddingHorizontal: 10, paddingVertical: 5 }
});
