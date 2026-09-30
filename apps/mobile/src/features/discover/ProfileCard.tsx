import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { LinearGradient } from "expo-linear-gradient";
import { Ionicons } from "@expo/vector-icons";
import { Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { mediaUrl } from "../../lib/api/config";
import type { PublicProfile } from "../../lib/api/types";
import { formatDistance } from "../../lib/format";

type Props = {
  profile: PublicProfile;
  onOpenDetails: () => void;
};

/** Full-bleed photo card. Tap the left/right side to browse photos. */
export function ProfileCard({ profile, onOpenDetails }: Props) {
  const { radii, colors } = useTheme();
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const photo = photos[Math.min(index, photos.length - 1)];
  const distance = formatDistance(profile.distanceKm);

  return (
    <View style={[styles.card, { borderRadius: radii.xl, backgroundColor: colors.surfaceMuted }]} testID="profile-card">
      {photo ? (
        <Image
          accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${profile.firstName}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          recyclingKey={photo.id}
          source={{ uri: mediaUrl(photo.url) }}
          style={StyleSheet.absoluteFill}
          transition={120}
        />
      ) : null}
      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" onPress={() => setIndex((i) => Math.max(0, i - 1))} style={styles.tapZone} />
        <Pressable accessibilityLabel="Photo suivante" onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))} style={styles.tapZone} />
      </View>
      {photos.length > 1 ? (
        <View pointerEvents="none" style={styles.indicators}>
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.indicator, { backgroundColor: i === index ? "#fff" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}
      <LinearGradient colors={["transparent", "rgba(10,6,8,0.85)"]} pointerEvents="box-none" style={styles.info}>
        <Pressable accessibilityHint="Ouvre le profil complet" accessibilityRole="button" onPress={onOpenDetails} style={styles.infoPress} testID="profile-card-details">
          <View style={styles.infoText}>
            <Text style={styles.white} variant="title">
              {profile.firstName}, {profile.age}
            </Text>
            {profile.jobTitle ? (
              <Text style={styles.whiteMuted} variant="body">
                {profile.jobTitle}
              </Text>
            ) : null}
            <View style={styles.meta}>
              {distance ? <Meta icon="location-outline" label={distance} /> : profile.city ? <Meta icon="location-outline" label={profile.city} /> : null}
              {profile.sharedInterests > 0 ? (
                <Meta icon="sparkles-outline" label={`${profile.sharedInterests} intérêt${profile.sharedInterests > 1 ? "s" : ""} en commun`} />
              ) : null}
            </View>
          </View>
          <Ionicons color="#fff" name="chevron-up-circle-outline" size={30} />
        </Pressable>
      </LinearGradient>
    </View>
  );
}

function Meta({ icon, label }: { icon: keyof typeof Ionicons.glyphMap; label: string }) {
  return (
    <View style={styles.metaItem}>
      <Ionicons color="#fff" name={icon} size={14} />
      <Text style={styles.whiteMuted} variant="caption">
        {label}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, overflow: "hidden" },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: "30%" },
  tapZone: { flex: 1 },
  indicators: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  indicator: { flex: 1, height: 3, borderRadius: 2 },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, paddingTop: 80 },
  infoPress: { flexDirection: "row", alignItems: "flex-end", padding: 20, gap: 12 },
  infoText: { flex: 1, gap: 4 },
  white: { color: "#fff" },
  whiteMuted: { color: "rgba(255,255,255,0.88)" },
  meta: { flexDirection: "row", flexWrap: "wrap", gap: 12, marginTop: 4 },
  metaItem: { flexDirection: "row", alignItems: "center", gap: 4 }
});
