import React from "react";
import { Image, ScrollView, View } from "react-native";
import { resolvePhotoUrl } from "../api/client";
import type { MyProfile } from "../api/profile";
import { useInterests } from "../hooks/useProfileData";
import { GENDER_LABELS, formatNameAge } from "../lib/format";
import { sortPhotos } from "../lib/photos";
import { Badge } from "../shared/ui/Badge";
import { Card } from "../shared/ui/Card";
import { Text } from "../shared/ui/Text";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";

const makeStyles = (t: Theme) => ({
  card: { padding: 0, overflow: "hidden" as const },
  photo: { width: "100%" as const, aspectRatio: 4 / 5, backgroundColor: t.colors.surfaceMuted },
  placeholder: { alignItems: "center" as const, justifyContent: "center" as const },
  body: { padding: t.spacing.lg, gap: t.spacing.sm },
  chips: { flexDirection: "row" as const, flexWrap: "wrap" as const, gap: t.spacing.xs },
  strip: { gap: t.spacing.xs, paddingHorizontal: t.spacing.lg, paddingBottom: t.spacing.md },
  thumb: { width: 56, height: 70, borderRadius: t.radii.xs, backgroundColor: t.colors.surfaceMuted }
});

/** The profile as other people see it. */
export function ProfilePreviewCard({ profile }: { profile: MyProfile }) {
  const styles = useThemedStyles(makeStyles);
  const interests = useInterests();
  const photos = sortPhotos(profile.photos);
  const main = photos[0];
  const labels = new Map((interests.data ?? []).map((interest) => [interest.slug, interest.label]));

  return (
    <Card
      accessibilityLabel={`Your profile preview. ${formatNameAge(profile.firstName, profile.showAge ? profile.age : null)}`}
      style={styles.card}
      testID="profile-preview"
    >
      {main ? (
        <Image
          accessibilityIgnoresInvertColors
          accessibilityLabel="Your main photo"
          source={{ uri: resolvePhotoUrl(main.url) }}
          style={styles.photo}
        />
      ) : (
        <View style={[styles.photo, styles.placeholder]}>
          <Text tone="muted">No photo yet</Text>
        </View>
      )}
      {photos.length > 1 ? (
        <ScrollView contentContainerStyle={styles.strip} horizontal showsHorizontalScrollIndicator={false}>
          {photos.slice(1).map((photo) => (
            <Image
              accessibilityIgnoresInvertColors
              key={photo.id}
              source={{ uri: resolvePhotoUrl(photo.url) }}
              style={styles.thumb}
            />
          ))}
        </ScrollView>
      ) : null}
      <View style={styles.body}>
        <Text variant="heading" weight="bold">
          {formatNameAge(profile.firstName, profile.showAge ? profile.age : null)}
        </Text>
        <Text tone="muted">
          {[GENDER_LABELS[profile.gender], profile.city].filter(Boolean).join(" · ")}
        </Text>
        {profile.bio ? <Text>{profile.bio}</Text> : <Text tone="subtle">You have not written a bio yet.</Text>}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.map((slug) => (
              <Badge key={slug} label={labels.get(slug) ?? slug} size="sm" variant="primary" />
            ))}
          </View>
        ) : null}
        {!profile.discoverable ? <Badge label="Profile paused" size="sm" variant="warning" /> : null}
      </View>
    </Card>
  );
}
