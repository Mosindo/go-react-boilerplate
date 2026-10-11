import { View } from 'react-native';
import type { PublicProfile } from '../../api';
import { fr } from '../../i18n';
import { formatDistance } from '../../lib/format';
import { useStyles } from '../../theme';
import { Chip, Text } from '../../ui';
import { PhotoCarousel } from './PhotoCarousel';

interface ProfileCardProps {
  profile: PublicProfile;
  onOpen: () => void;
}

export function ProfileCard({ profile, onOpen }: ProfileCardProps) {
  const styles = useStyles((t) => ({
    card: {
      flex: 1,
      borderRadius: t.radius.xl,
      overflow: 'hidden',
      backgroundColor: t.colors.photoPlaceholder,
      shadowColor: '#3A1D10',
      shadowOpacity: 0.18,
      shadowRadius: 16,
      shadowOffset: { width: 0, height: 8 },
      elevation: 4,
    },
    info: {
      position: 'absolute',
      left: 0,
      right: 0,
      bottom: 0,
      paddingHorizontal: t.spacing.lg,
      paddingTop: t.spacing.xxl,
      paddingBottom: t.spacing.lg,
      gap: t.spacing.xs,
      backgroundColor: 'rgba(20,10,6,0.52)',
    },
    tags: {
      flexDirection: 'row',
      gap: t.spacing.xs,
      flexWrap: 'wrap',
      marginTop: t.spacing.xs,
      overflow: 'hidden',
      maxHeight: 32,
    },
  }));
  const distance = formatDistance(profile.distanceKm);
  const where = [profile.city, distance].filter(Boolean).join(' · ');
  return (
    <View style={styles.card}>
      <PhotoCarousel
        photos={profile.photos}
        name={profile.firstName}
        onCenterPress={onOpen}
        style={{ flex: 1 }}
      />
      <View style={styles.info} pointerEvents="none">
        <Text variant="title" color="onPhoto" accessible={false}>
          {profile.firstName}, {profile.age}
        </Text>
        {where ? (
          <Text color="onPhoto" accessible={false}>
            {where}
          </Text>
        ) : null}
        {profile.likedYou ? (
          <Text variant="small" color="onPhoto" weight="700" accessible={false}>
            {fr.discover.likedYou}
          </Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.tags}>
            {profile.interests.slice(0, 3).map((interest) => (
              <Chip key={interest.id} label={interest.label} />
            ))}
          </View>
        ) : null}
      </View>
    </View>
  );
}
