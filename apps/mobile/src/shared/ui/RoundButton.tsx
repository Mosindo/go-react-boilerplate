import React from 'react';
import { Pressable, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useTheme } from '../../theme';

export function RoundButton({
  icon,
  label,
  color,
  filled,
  disabled,
  onPress,
  testID,
  size = 64,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  color: string;
  filled?: boolean;
  disabled?: boolean;
  onPress: () => void;
  testID: string;
  size?: number;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: filled ? color : colors.surface,
          borderColor: color,
          opacity: disabled ? 0.4 : pressed ? 0.85 : 1,
        },
      ]}
    >
      <Ionicons name={icon} size={size * 0.46} color={filled ? colors.onPrimary : color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  round: { alignItems: 'center', justifyContent: 'center', borderWidth: 2 },
});
