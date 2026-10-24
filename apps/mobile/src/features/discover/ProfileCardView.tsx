import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { Card } from "../../api/types";
import { radius, spacing } from "../../theme/theme";
import { Icon, Photo, Text } from "../../ui";

type Props = { card: Card; onOpenProfile: () => void };

/** Full-bleed photo card: tap the right/left side to browse photos, tap (i) for the full profile. */
export function ProfileCardView({ card, onOpenProfile }: Props) {
  const [index, setIndex] = useState(0);
  const photo = card.photos[Math.min(index, card.photos.length - 1)];
  const go = (delta: number) =>
    setIndex((i) => Math.max(0, Math.min(card.photos.length - 1, i + delta)));
  const subtitle = [card.city, card.distanceKm ? `à ${card.distanceKm} km` : null]
    .filter(Boolean)
    .join(" · ");

  return (
    <View style={styles.card} accessibilityLabel={`Profil de ${card.firstName}, ${card.age} ans`}>
      <Photo
        photoId={photo?.id}
        style={StyleSheet.absoluteFill}
        accessibilityLabel={`Photo de ${card.firstName}`}
      />
      <View style={styles.zones} pointerEvents="box-none">
        <Pressable
          testID="card-prev-photo"
          accessibilityLabel="Photo précédente"
          style={styles.zone}
          onPress={() => go(-1)}
        />
        <Pressable
          testID="card-next-photo"
          accessibilityLabel="Photo suivante"
          style={styles.zone}
          onPress={() => go(1)}
        />
      </View>
      {card.photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, { opacity: i === index ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}
      <LinearGradient
        colors={["transparent", "rgba(20,10,28,0.85)"]}
        style={styles.gradient}
        pointerEvents="none"
      />
      <View style={styles.info} pointerEvents="box-none">
        <View style={{ flex: 1 }} pointerEvents="none">
          <Text variant="title" color="#FFFFFF" testID="card-name">
            {card.firstName}, {card.age}
          </Text>
          {subtitle ? (
            <Text color="#FFFFFF" style={styles.subtitle}>
              {subtitle}
            </Text>
          ) : null}
          {card.bio ? (
            <Text color="#FFFFFF" numberOfLines={2} style={styles.bio}>
              {card.bio}
            </Text>
          ) : null}
        </View>
        <Pressable
          testID="card-open-profile"
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil complet de ${card.firstName}`}
          onPress={onOpenProfile}
          style={styles.infoButton}
        >
          <Icon name="information" size={26} color="#FFFFFF" />
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radius.lg, overflow: "hidden", backgroundColor: "#2C2434" },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  zone: { flex: 1 },
  dots: {
    position: "absolute",
    top: spacing.md,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    gap: 4,
  },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: "#FFFFFF" },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, height: "45%" },
  info: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    bottom: spacing.lg,
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.md,
  },
  subtitle: { fontSize: 15, marginTop: 2 },
  bio: { marginTop: spacing.xs, opacity: 0.92 },
  infoButton: {
    width: 44,
    height: 44,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: 22,
    backgroundColor: "rgba(255,255,255,0.22)",
  },
});
