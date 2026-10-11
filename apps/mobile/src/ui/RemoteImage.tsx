import { useEffect, useState } from 'react';
import { Image, View } from 'react-native';
import type { ImageStyle, StyleProp, ViewStyle } from 'react-native';
import { useStyles } from '../theme';
import { Text } from './Text';

interface RemoteImageProps {
  uri: string | null | undefined;
  style?: StyleProp<ViewStyle>;
  /** Shown when there is no image or it fails to load (expired signature, offline). */
  fallbackText?: string;
  accessibilityLabel?: string;
}

/** Cover-fit remote image with a warm placeholder and graceful failure. */
export function RemoteImage({ uri, style, fallbackText, accessibilityLabel }: RemoteImageProps) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [uri]);
  const styles = useStyles((t) => ({
    box: {
      backgroundColor: t.colors.photoPlaceholder,
      overflow: 'hidden',
      alignItems: 'center',
      justifyContent: 'center',
    },
    image: { ...({ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 } as const) },
  }));
  return (
    <View
      style={[styles.box, style]}
      accessible={!!accessibilityLabel}
      accessibilityRole="image"
      accessibilityLabel={accessibilityLabel}
    >
      {fallbackText ? (
        <Text variant="title" color="textMuted" accessible={false}>
          {fallbackText}
        </Text>
      ) : null}
      {uri && !failed ? (
        <Image
          source={{ uri }}
          resizeMode="cover"
          style={styles.image as ImageStyle}
          onError={() => setFailed(true)}
        />
      ) : null}
    </View>
  );
}
