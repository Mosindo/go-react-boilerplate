import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { PublicProfile } from "../../api/types";
import { radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Icon, RemotePhoto, Text } from "../../ui";
import { distanceLabel } from "../../utils/format";

/** Card face: tap left/right to browse photos, essentials overlaid. */
export function ProfileCard({ profile, onOpen }: { profile: PublicProfile; onOpen: () => void }) {
  const { colors } = useTheme();
  const [photoIndex, setPhotoIndex] = useState(0);
  const photo = profile.photos[photoIndex] ?? profile.photos[0];
  const distance = distanceLabel(profile.distanceKm);
  const count = profile.photos.length;

  return (
    <View style={[styles.card, { backgroundColor: colors.surfaceMuted, shadowColor: colors.shadow }]}>
      <RemotePhoto accessibilityLabel={`Photo de ${profile.firstName}`} photo={photo} priority="high" style={StyleSheet.absoluteFill} />
      {count > 1 ? (
        <View style={styles.bars}>
          {profile.photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, { backgroundColor: i === photoIndex ? "#fff" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}
      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" onPress={() => setPhotoIndex((i) => Math.max(0, i - 1))} style={styles.flex} />
        <Pressable accessibilityLabel="Photo suivante" onPress={() => setPhotoIndex((i) => Math.min(count - 1, i + 1))} style={styles.flex} />
      </View>
      <Pressable accessibilityHint="Ouvre le profil complet" accessibilityRole="button" onPress={onOpen} style={styles.info}>
        <LinearGradient colors={["rgba(15,8,11,0)", "rgba(15,8,11,0.55)", "rgba(15,8,11,0.88)"]} locations={[0, 0.4, 1]} style={styles.fade} />
        <View style={styles.infoInner}>
          <View style={styles.nameRow}>
            <Text style={styles.name} variant="title">
              {profile.firstName}, {profile.age}
            </Text>
            <Icon color="#fff" name="information-circle-outline" size={26} />
          </View>
          {profile.jobTitle || distance ? (
            <Text style={styles.meta} variant="label">
              {[profile.jobTitle, distance].filter(Boolean).join(" · ")}
            </Text>
          ) : null}
          {profile.interests.length ? (
            <View style={styles.chips}>
              {profile.interests.slice(0, 3).map((i) => (
                <View key={i.slug} style={styles.chip}>
                  <Text style={styles.chipText} variant="caption" weight="600">
                    {i.label}
                  </Text>
                </View>
              ))}
            </View>
          ) : null}
        </View>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    flex: 1,
    borderRadius: radii.xl,
    overflow: "hidden",
    shadowOpacity: 0.18,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 12 },
    elevation: 6
  },
  bars: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4, zIndex: 2 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: "35%" },
  flex: { flex: 1 },
  info: { position: "absolute", left: 0, right: 0, bottom: 0 },
  fade: { position: "absolute", left: 0, right: 0, bottom: 0, top: -90 },
  infoInner: { padding: spacing.lg, paddingTop: spacing.xl, gap: spacing.xs },
  nameRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  name: { color: "#fff", flexShrink: 1 },
  meta: { color: "rgba(255,255,255,0.9)" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: 2 },
  chip: { backgroundColor: "rgba(255,255,255,0.18)", borderRadius: radii.pill, paddingHorizontal: 10, paddingVertical: 4 },
  chipText: { color: "#fff" }
});
