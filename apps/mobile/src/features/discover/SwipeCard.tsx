import { Ionicons } from "@expo/vector-icons";
import { LinearGradient } from "expo-linear-gradient";
import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Photo, Text } from "../../ui";
import { formatDistance } from "../../utils/format";

type Props = { profile: PublicProfile; onOpen: () => void };

/** Visual card; gestures are handled by the deck. Tap sides to browse photos. */
export function SwipeCard({ profile, onOpen }: Props) {
  const { colors, radii } = useTheme();
  const [index, setIndex] = useState(0);
  const photo = profile.photos[index] ?? profile.photos[0];
  const distance = formatDistance(profile.distanceKm);
  const subtitle = [profile.jobTitle, profile.city, distance].filter(Boolean).join(" · ");
  const shared = profile.interests.filter((i) => profile.sharedInterests.includes(i.slug)).slice(0, 3);

  return (
    <View
      style={[styles.card, { borderRadius: radii.xl, backgroundColor: colors.surfaceAlt, shadowColor: colors.shadow }]}
    >
      <Photo
        url={photo?.url}
        style={StyleSheet.absoluteFill}
        accessibilityLabel={`Photo ${index + 1} de ${profile.firstName}`}
      />
      {profile.photos.length > 1 ? (
        <View style={styles.bars}>
          {profile.photos.map((p, i) => (
            <View
              key={p.id}
              style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]}
            />
          ))}
        </View>
      ) : null}
      <View style={styles.tapZones}>
        <Pressable
          accessibilityLabel="Photo précédente"
          style={styles.flex}
          onPress={() => setIndex((i) => Math.max(0, i - 1))}
        />
        <Pressable
          accessibilityLabel="Photo suivante"
          style={styles.flex}
          onPress={() => setIndex((i) => Math.min(profile.photos.length - 1, i + 1))}
        />
      </View>
      <LinearGradient colors={["transparent", "rgba(15,8,12,0.88)"]} style={styles.info} pointerEvents="box-none">
        <Pressable
          onPress={onOpen}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil complet de ${profile.firstName}`}
          testID="card-open"
        >
          <View style={styles.nameRow}>
            <Text variant="title" style={styles.white} testID="card-name">
              {profile.firstName}, {profile.age}
            </Text>
            <Ionicons name="information-circle-outline" size={26} color="#FFFFFF" />
          </View>
          {subtitle ? (
            <Text style={styles.whiteMuted} numberOfLines={1}>
              {subtitle}
            </Text>
          ) : null}
          {shared.length > 0 ? (
            <View style={styles.shared}>
              {shared.map((i) => (
                <View key={i.slug} style={styles.sharedChip}>
                  <Text variant="caption" weight="semibold" style={styles.white}>
                    {i.label}
                  </Text>
                </View>
              ))}
            </View>
          ) : profile.bio ? (
            <Text style={styles.whiteMuted} numberOfLines={2}>
              {profile.bio}
            </Text>
          ) : null}
          {profile.isDemo ? (
            <Text variant="caption" style={styles.whiteMuted}>
              Profil de démonstration
            </Text>
          ) : null}
        </Pressable>
      </LinearGradient>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    overflow: "hidden",
    shadowOpacity: 0.18,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 10 },
    elevation: 6
  },
  bars: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4, zIndex: 2 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: "35%" },
  flex: { flex: 1 },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    paddingHorizontal: 20,
    paddingTop: 80,
    paddingBottom: 22
  },
  nameRow: { flexDirection: "row", alignItems: "center", gap: 8 },
  white: { color: "#FFFFFF" },
  whiteMuted: { color: "rgba(255,255,255,0.86)", marginTop: 2 },
  shared: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: 10 },
  sharedChip: {
    backgroundColor: "rgba(255,255,255,0.18)",
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 999
  }
});
