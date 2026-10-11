import { useEffect, useState } from 'react';
import { Image, Pressable, View } from 'react-native';
import type { StyleProp, ViewStyle } from 'react-native';
import { mediaUrl } from '../../api';
import type { Photo } from '../../api';
import { fr } from '../../i18n';
import { initials } from '../../lib/format';
import { useStyles } from '../../theme';
import { RemoteImage } from '../../ui';

interface PhotoCarouselProps {
  photos: Photo[];
  name: string;
  /** Tap on the middle third; when omitted the two halves just navigate photos. */
  onCenterPress?: () => void;
  style?: StyleProp<ViewStyle>;
}

/** Photo viewer with tap zones (left/right = previous/next) and progress dots. */
export function PhotoCarousel({ photos, name, onCenterPress, style }: PhotoCarouselProps) {
  const [index, setIndex] = useState(0);
  const count = photos.length;
  const current = Math.min(index, Math.max(0, count - 1));
  const styles = useStyles((t) => ({
    root: { overflow: 'hidden', backgroundColor: t.colors.photoPlaceholder },
    fill: { position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 },
    zones: { position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, flexDirection: 'row' },
    dots: { position: 'absolute', top: 10, left: 12, right: 12, flexDirection: 'row', gap: 4 },
    dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: 'rgba(255,255,255,0.4)' },
    dotActive: { backgroundColor: '#FFFFFF' },
  }));

  // Warm the next photo so tapping through feels instant.
  useEffect(() => {
    const next = photos[current + 1];
    if (next) void Image.prefetch(mediaUrl(next.url)).catch(() => undefined);
  }, [photos, current]);

  const go = (delta: number) => setIndex(Math.min(count - 1, Math.max(0, current + delta)));
  const photo = photos[current];

  return (
    <View style={[styles.root, style]}>
      <RemoteImage
        uri={photo ? mediaUrl(photo.url) : null}
        fallbackText={initials(name)}
        style={styles.fill}
      />
      {count > 1 ? (
        <View
          style={styles.dots}
          pointerEvents="none"
          accessible
          accessibilityLabel={fr.discover.photoPosition(current + 1, count)}
        >
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, i === current && styles.dotActive]} />
          ))}
        </View>
      ) : null}
      <View style={styles.zones}>
        <Pressable
          style={{ flex: onCenterPress ? 1 : 1 }}
          accessibilityRole="button"
          accessibilityLabel={fr.discover.previousPhoto}
          disabled={count < 2}
          onPress={() => go(-1)}
        />
        {onCenterPress ? (
          <Pressable
            style={{ flex: 1.2 }}
            accessibilityRole="button"
            accessibilityLabel={fr.discover.openProfile(name)}
            onPress={onCenterPress}
          />
        ) : null}
        <Pressable
          style={{ flex: 1 }}
          accessibilityRole="button"
          accessibilityLabel={fr.discover.nextPhoto}
          disabled={count < 2}
          onPress={() => go(1)}
        />
      </View>
    </View>
  );
}
