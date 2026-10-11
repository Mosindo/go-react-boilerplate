import { Pressable } from 'react-native';
import type { StyleProp, ViewStyle } from 'react-native';
import { TOUCH_TARGET, useStyles } from '../theme';
import { Text } from './Text';
import type { TextColor } from './Text';

interface IconButtonProps {
  glyph: string;
  accessibilityLabel: string;
  onPress: () => void;
  size?: number;
  color?: TextColor;
  tone?: 'plain' | 'filled' | 'outline' | 'primary' | 'photo';
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
}

export function IconButton({
  glyph,
  accessibilityLabel,
  onPress,
  size = TOUCH_TARGET,
  color = 'text',
  tone = 'plain',
  disabled,
  testID,
  style,
}: IconButtonProps) {
  const styles = useStyles((t) => ({
    base: { alignItems: 'center', justifyContent: 'center' },
    plain: {},
    filled: { backgroundColor: t.colors.surfaceAlt },
    outline: { backgroundColor: t.colors.surface, borderWidth: 1, borderColor: t.colors.border },
    primary: { backgroundColor: t.colors.primary },
    photo: { backgroundColor: 'rgba(0,0,0,0.38)' },
    pressed: { opacity: 0.7, transform: [{ scale: 0.94 }] },
    disabled: { opacity: 0.4 },
  }));
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      accessibilityState={{ disabled: !!disabled }}
      disabled={disabled}
      hitSlop={size < TOUCH_TARGET ? (TOUCH_TARGET - size) / 2 : 0}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        styles[tone],
        { width: size, height: size, borderRadius: size / 2 },
        pressed && styles.pressed,
        disabled && styles.disabled,
        style,
      ]}
    >
      <Text
        accessible={false}
        color={tone === 'primary' ? 'textOnPrimary' : tone === 'photo' ? 'onPhoto' : color}
        style={{
          fontSize: Math.round(size * 0.42),
          lineHeight: Math.round(size * 0.5),
          fontWeight: '700',
        }}
      >
        {glyph}
      </Text>
    </Pressable>
  );
}
