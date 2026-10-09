import React, { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import type { Interest, Photo } from '../api/types';
import { fr } from '../i18n/fr';
import { radius, spacing, useTheme } from '../theme';
import { AuthImage, Chip, Text } from './ui';

export type ProfileViewData = {
  firstName: string;
  age: number;
  city: string;
  distanceKm?: number | null;
  bio: string;
  interests: Interest[];
  photos: Photo[];
};

/** Full profile: photo carousel (tap zones) + details. Used for others and for "my preview". */
export function ProfileView({ profile }: { profile: ProfileViewData }) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const photos = profile.photos;
  const current = photos[Math.min(index, Math.max(photos.length - 1, 0))];
  const place = [
    profile.city,
    profile.distanceKm != null ? fr.common.kmAway(profile.distanceKm) : null,
  ]
    .filter(Boolean)
    .join(' · ');
  return (
    <View testID="profile-view">
      <View style={[styles.photoWrap, { backgroundColor: colors.surfaceAlt }]}>
        <AuthImage
          path={current?.url}
          style={StyleSheet.absoluteFill}
          resizeMode="cover"
          accessibilityLabel={fr.photos.photoOf(
            profile.firstName,
            Math.min(index + 1, photos.length),
            photos.length,
          )}
          fallback={<Ionicons name="person" size={64} color={colors.textMuted} />}
          testID="profile-photo"
        />
        {photos.length > 1 ? (
          <>
            <View style={styles.dots} pointerEvents="none">
              {photos.map((p, i) => (
                <View
                  key={p.id}
                  style={[
                    styles.dot,
                    { backgroundColor: i === index ? '#FFFFFF' : 'rgba(255,255,255,0.4)' },
                  ]}
                />
              ))}
            </View>
            <View style={styles.zones}>
              <Pressable
                testID="profile-prev"
                accessibilityRole="button"
                accessibilityLabel={fr.discover.previousPhoto}
                style={styles.zone}
                onPress={() => setIndex((i) => Math.max(0, i - 1))}
              />
              <Pressable
                testID="profile-next"
                accessibilityRole="button"
                accessibilityLabel={fr.discover.nextPhoto}
                style={styles.zone}
                onPress={() => setIndex((i) => Math.min(photos.length - 1, i + 1))}
              />
            </View>
          </>
        ) : null}
      </View>

      <View style={styles.details}>
        <Text variant="display" accessibilityRole="header" testID="profile-name">
          {profile.firstName}, {profile.age}
        </Text>
        {place ? (
          <View style={styles.placeRow}>
            <Ionicons name="location-outline" size={16} color={colors.textMuted} />
            <Text tone="muted" style={styles.place}>
              {place}
            </Text>
          </View>
        ) : null}

        <Text variant="heading" style={styles.section}>
          {fr.profile.about}
        </Text>
        <Text tone={profile.bio ? 'default' : 'muted'}>{profile.bio || fr.profile.noBio}</Text>

        {profile.interests.length > 0 ? (
          <>
            <Text variant="heading" style={styles.section}>
              {fr.interests.title}
            </Text>
            <View style={styles.chips}>
              {profile.interests.map((i) => (
                <Chip key={i.slug} label={i.label} />
              ))}
            </View>
          </>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  photoWrap: {
    width: '100%',
    aspectRatio: 3 / 4,
    maxHeight: 640,
    overflow: 'hidden',
    borderRadius: radius.lg,
  },
  dots: {
    position: 'absolute',
    top: spacing.sm,
    left: spacing.md,
    right: spacing.md,
    flexDirection: 'row',
    gap: 4,
  },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: 'row' },
  zone: { flex: 1 },
  details: { paddingTop: spacing.lg },
  placeRow: { flexDirection: 'row', alignItems: 'center', marginTop: spacing.xs },
  place: { marginLeft: spacing.xs },
  section: { marginTop: spacing.xl, marginBottom: spacing.sm },
  chips: { flexDirection: 'row', flexWrap: 'wrap' },
});
