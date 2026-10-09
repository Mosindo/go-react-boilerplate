import React from 'react';
import { StyleSheet } from 'react-native';
import { useTheme } from '../../theme';
import { AuthImage } from './AuthImage';
import { Text } from './Text';

function AvatarImpl({
  name,
  path,
  size = 48,
  testID,
}: {
  name: string;
  path: string | null | undefined;
  size?: number;
  testID?: string;
}) {
  const { colors } = useTheme();
  const initial = name.trim().charAt(0).toUpperCase();
  return (
    <AuthImage
      testID={testID}
      path={path}
      accessibilityLabel={name}
      style={[styles.img, { width: size, height: size, borderRadius: size / 2 }]}
      fallback={
        <Text variant="bodyStrong" style={{ color: colors.textMuted, fontSize: size / 2.4 }}>
          {initial}
        </Text>
      }
    />
  );
}

const styles = StyleSheet.create({ img: { overflow: 'hidden' } });

export const Avatar = React.memo(AvatarImpl);
