import React from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import type { OwnProfile, PublicProfile } from "../api/types";
import { distanceLabel } from "../lib/format";
import { spacing } from "../theme";
import { AuthImage } from "./AuthImage";
import { Chip } from "./Chip";
import { Text } from "./Text";

type ViewableProfile = Pick<PublicProfile, "firstName" | "age" | "city" | "bio" | "interests" | "photos"> & {
  distanceKm?: number | null;
};

export function ProfileView({ profile }: { profile: ViewableProfile | OwnProfile }) {
  const { width } = useWindowDimensions();
  const photos = [...profile.photos].sort((a, b) => a.position - b.position);
  const distance = "distanceKm" in profile ? distanceLabel(profile.distanceKm) : null;
  return (
    <View style={{ gap: spacing.lg }}>
      {photos.length > 0 ? (
        <ScrollView horizontal pagingEnabled showsHorizontalScrollIndicator={false} accessibilityLabel="Photos">
          {photos.map((photo, i) => (
            <AuthImage key={photo.id} url={photo.url} style={{ width, height: width * 1.2 }} label={`${profile.firstName} photo ${i + 1}`} />
          ))}
        </ScrollView>
      ) : (
        <AuthImage url={null} style={{ width, height: width * 0.8 }} fallbackInitial={profile.firstName} />
      )}
      <View style={styles.body}>
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.city || distance ? <Text tone="muted">{[profile.city, distance].filter(Boolean).join(" · ")}</Text> : null}
        {profile.bio ? <Text>{profile.bio}</Text> : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.map((interest) => (
              <Chip key={interest.id} label={interest.label} />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  body: { paddingHorizontal: spacing.lg, gap: spacing.md, paddingBottom: spacing.xxl },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
