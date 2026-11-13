import { Ionicons } from "@expo/vector-icons";
import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Chip, Photo, Text } from "../../ui";
import { formatDistance, goalLabels } from "../../utils/format";

type Props = { profile: PublicProfile; footer?: React.ReactNode };

/** Full profile layout shared by the profile detail and self preview. */
export function ProfileView({ profile, footer }: Props) {
  const { colors, spacing, radii } = useTheme();
  const { width } = useWindowDimensions();
  const pageWidth = Math.min(width, 640);
  const [index, setIndex] = useState(0);
  const distance = formatDistance(profile.distanceKm);

  const facts: { icon: keyof typeof Ionicons.glyphMap; text: string }[] = [];
  if (profile.jobTitle) facts.push({ icon: "briefcase-outline", text: profile.jobTitle });
  if (profile.relationshipGoal)
    facts.push({ icon: "heart-circle-outline", text: goalLabels[profile.relationshipGoal] });
  if (profile.heightCm) facts.push({ icon: "resize-outline", text: `${profile.heightCm} cm` });
  const place = [profile.city, distance].filter(Boolean).join(" · ");
  if (place) facts.push({ icon: "location-outline", text: place });

  return (
    <ScrollView contentContainerStyle={{ paddingBottom: spacing.xxxl }} testID="profile-view">
      <View style={{ width: pageWidth, alignSelf: "center" }}>
        <ScrollView
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          onMomentumScrollEnd={(e) => setIndex(Math.round(e.nativeEvent.contentOffset.x / pageWidth))}
          accessibilityLabel={`${profile.photos.length} photos`}
        >
          {profile.photos.map((photo, i) => (
            <Photo
              key={photo.id}
              url={photo.url}
              style={{ width: pageWidth, aspectRatio: 4 / 5 }}
              accessibilityLabel={`Photo ${i + 1} de ${profile.firstName}`}
            />
          ))}
        </ScrollView>
        {profile.photos.length > 1 ? (
          <View style={styles.dots}>
            {profile.photos.map((p, i) => (
              <View
                key={p.id}
                style={[styles.dot, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.45)" }]}
              />
            ))}
          </View>
        ) : null}
      </View>

      <View style={{ padding: spacing.lg, gap: spacing.lg, maxWidth: 640, width: "100%", alignSelf: "center" }}>
        <View style={{ gap: 4 }}>
          <View style={styles.nameRow}>
            <Text variant="title" accessibilityRole="header">
              {profile.firstName}, {profile.age}
            </Text>
            {profile.isDemo ? (
              <View style={[styles.demo, { backgroundColor: colors.accentSoft, borderRadius: radii.sm }]}>
                <Text variant="caption" weight="semibold" tone="accent">
                  Profil de démonstration
                </Text>
              </View>
            ) : null}
          </View>
          {facts.map((f) => (
            <View key={f.text} style={styles.fact}>
              <Ionicons name={f.icon} size={18} color={colors.textMuted} />
              <Text tone="muted">{f.text}</Text>
            </View>
          ))}
        </View>

        {profile.bio ? (
          <View
            style={[
              styles.card,
              { backgroundColor: colors.surface, borderRadius: radii.lg, borderColor: colors.border }
            ]}
          >
            <Text variant="label" weight="semibold" tone="muted">
              À propos
            </Text>
            <Text>{profile.bio}</Text>
          </View>
        ) : null}

        {profile.interests.length > 0 ? (
          <View style={{ gap: spacing.sm }}>
            <Text variant="label" weight="semibold" tone="muted">
              Centres d’intérêt
              {profile.sharedInterests.length > 0 ? ` · ${profile.sharedInterests.length} en commun` : ""}
            </Text>
            <View style={styles.chips}>
              {profile.interests.map((i) => (
                <Chip key={i.slug} label={i.label} highlighted={profile.sharedInterests.includes(i.slug)} />
              ))}
            </View>
          </View>
        ) : null}
        {footer}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  dots: { position: "absolute", top: 12, left: 0, right: 0, flexDirection: "row", justifyContent: "center", gap: 6 },
  dot: { width: 28, height: 4, borderRadius: 2 },
  nameRow: { flexDirection: "row", alignItems: "center", flexWrap: "wrap", gap: 10 },
  demo: { paddingHorizontal: 8, paddingVertical: 3 },
  fact: { flexDirection: "row", alignItems: "center", gap: 8 },
  card: { padding: 16, gap: 6, borderWidth: StyleSheet.hairlineWidth },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
