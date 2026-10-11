import { View } from 'react-native';
import type { PublicProfile } from '../../../api';
import { fr } from '../../../i18n';
import { formatDistance } from '../../../lib/format';
import { useStyles } from '../../../theme';
import { Chip, Text } from '../../../ui';
import { PhotoCarousel } from '../../discover/PhotoCarousel';

/** Full public profile: all photos, identity line, distance, bio and interests. */
export function ProfileView({ profile }: { profile: PublicProfile }) {
  const styles = useStyles((t) => ({
    wrap: { gap: t.spacing.lg },
    photo: { width: '100%', aspectRatio: 3 / 4, borderRadius: t.radius.xl },
    section: { gap: t.spacing.sm },
    chips: { flexDirection: 'row', flexWrap: 'wrap', gap: t.spacing.sm },
    meta: { flexDirection: 'row', flexWrap: 'wrap', gap: t.spacing.sm },
  }));
  const distance = formatDistance(profile.distanceKm);
  return (
    <View style={styles.wrap}>
      <PhotoCarousel photos={profile.photos} name={profile.firstName} style={styles.photo} />
      <View style={styles.section}>
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        <View style={styles.meta}>
          <Chip label={fr.gender.single[profile.gender]} />
          {profile.city ? <Chip label={profile.city} /> : null}
          {distance ? <Chip label={distance} /> : null}
        </View>
        {profile.likedYou ? (
          <Text color="primary" weight="700">
            {fr.discover.likedYou}
          </Text>
        ) : null}
      </View>
      {profile.bio ? (
        <View style={styles.section}>
          <Text variant="label" color="textMuted">
            {fr.profile.about}
          </Text>
          <Text>{profile.bio}</Text>
        </View>
      ) : null}
      {profile.interests.length > 0 ? (
        <View style={styles.section}>
          <Text variant="label" color="textMuted">
            {fr.profile.interests}
          </Text>
          <View style={styles.chips}>
            {profile.interests.map((interest) => (
              <Chip key={interest.id} label={interest.label} />
            ))}
          </View>
        </View>
      ) : null}
    </View>
  );
}
