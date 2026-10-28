import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { distanceLabel } from "../lib/format";
import { nextPhotoIndex, photoTapDirection } from "../lib/swipe";
import { MIN_TOUCH, radii, spacing } from "../theme";
import { AuthImage } from "./AuthImage";
import { Chip } from "./Chip";
import { Text } from "./Text";

type Props = {
  profile: PublicProfile;
  onOpenProfile?: () => void;
  width: number;
};

export function ProfileCard({ profile, onOpenProfile, width }: Props) {
  const [index, setIndex] = useState(0);
  const photos = [...profile.photos].sort((a, b) => a.position - b.position);
  const photo = photos[index];
  const distance = distanceLabel(profile.distanceKm);

  return (
    <View style={styles.card} testID={`discover-card-${profile.userId}`}>
      <AuthImage url={photo?.url} style={StyleSheet.absoluteFill} label={`${profile.firstName} photo ${index + 1}`} fallbackInitial={profile.firstName} />

      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, { opacity: i === index ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}

      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Next or previous photo"
        style={StyleSheet.absoluteFill}
        onPress={(e) => setIndex((i) => nextPhotoIndex(i, photoTapDirection(e.nativeEvent.locationX, width), photos.length))}
      />

      <View style={styles.info} pointerEvents="box-none">
        <View style={styles.infoRow} pointerEvents="box-none">
          <View style={{ flex: 1 }} pointerEvents="none">
            <Text variant="title" tone="inverse" style={styles.shadow}>
              {profile.firstName}, {profile.age}
            </Text>
            {profile.city || distance ? (
              <Text tone="inverse" style={styles.shadow}>
                {[profile.city, distance].filter(Boolean).join(" · ")}
              </Text>
            ) : null}
          </View>
          {onOpenProfile ? (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`View ${profile.firstName}'s full profile`}
              onPress={onOpenProfile}
              testID="discover-info"
              style={styles.infoButton}
            >
              <Text tone="inverse" style={{ fontWeight: "700", fontSize: 18 }}>
                i
              </Text>
            </Pressable>
          ) : null}
        </View>
        {profile.interests.length > 0 ? (
          <View style={styles.chips} pointerEvents="none">
            {profile.interests.slice(0, 4).map((interest) => (
              <Chip key={interest.id} label={interest.label} />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radii.lg, overflow: "hidden", backgroundColor: "#000" },
  dots: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: "#fff" },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, backgroundColor: "rgba(20,10,24,0.55)", gap: spacing.sm },
  infoRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  infoButton: { width: MIN_TOUCH, height: MIN_TOUCH, borderRadius: MIN_TOUCH / 2, borderWidth: 1.5, borderColor: "#fff", alignItems: "center", justifyContent: "center" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6 },
  shadow: { textShadowColor: "rgba(0,0,0,0.4)", textShadowRadius: 4 }
});
