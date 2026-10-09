import React from 'react';
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useTheme } from '../../theme';

export type IconButtonProps = {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress?: () => void;
  size?: number;
  color?: string;
  background?: string;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
  badge?: number;
};

function IconButtonImpl({
  icon,
  label,
  onPress,
  size = 24,
  color,
  background,
  disabled,
  testID,
  style,
  badge,
}: IconButtonProps) {
  const { colors } = useTheme();
  const dim = size + 20;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!disabled }}
      testID={testID}
      disabled={disabled}
      onPress={onPress}
      hitSlop={6}
      style={({ pressed }) => [
        styles.base,
        {
          width: dim,
          height: dim,
          borderRadius: dim / 2,
          backgroundColor: background ?? 'transparent',
        },
        pressed ? { opacity: 0.7 } : null,
        disabled ? { opacity: 0.4 } : null,
        style,
      ]}
    >
      <Ionicons name={icon} size={size} color={color ?? colors.text} />
      {badge && badge > 0 ? (
        <Ionicons name="ellipse" size={10} color={colors.primary} style={styles.dot} />
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: 'center', justifyContent: 'center' },
  dot: { position: 'absolute', top: 8, right: 8 },
});

export const IconButton = React.memo(IconButtonImpl);
