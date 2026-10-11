import { ActivityIndicator, Pressable } from 'react-native';
import type { StyleProp, ViewStyle } from 'react-native';
import { TOUCH_TARGET, useStyles, useTheme } from '../theme';
import { Text } from './Text';
import type { TextColor } from './Text';

export type ButtonVariant = 'primary' | 'secondary' | 'soft' | 'ghost' | 'danger';

interface ButtonProps {
  label: string;
  onPress: () => void;
  variant?: ButtonVariant;
  size?: 'md' | 'sm';
  loading?: boolean;
  disabled?: boolean;
  fullWidth?: boolean;
  accessibilityLabel?: string;
  accessibilityHint?: string;
  testID?: string;
  style?: StyleProp<ViewStyle>;
}

export function Button({
  label,
  onPress,
  variant = 'primary',
  size = 'md',
  loading = false,
  disabled = false,
  fullWidth = true,
  accessibilityLabel,
  accessibilityHint,
  testID,
  style,
}: ButtonProps) {
  const { colors } = useTheme();
  const styles = useStyles((t) => ({
    base: {
      borderRadius: t.radius.pill,
      alignItems: 'center',
      justifyContent: 'center',
      flexDirection: 'row',
      paddingHorizontal: t.spacing.xl,
      borderWidth: 1,
      borderColor: 'transparent',
    },
    md: { minHeight: 52 },
    sm: { minHeight: TOUCH_TARGET, paddingHorizontal: t.spacing.lg },
    primary: { backgroundColor: t.colors.primary },
    secondary: { backgroundColor: t.colors.secondary },
    soft: { backgroundColor: t.colors.primarySoft },
    ghost: { backgroundColor: 'transparent', borderColor: t.colors.border },
    danger: { backgroundColor: t.colors.dangerSoft, borderColor: t.colors.danger },
    disabled: { opacity: 0.5 },
    pressed: { opacity: 0.85, transform: [{ scale: 0.98 }] },
    fullWidth: { alignSelf: 'stretch' },
  }));
  const labelColor: TextColor = {
    primary: 'textOnPrimary',
    secondary: 'textOnSecondary',
    soft: 'primary',
    ghost: 'text',
    danger: 'danger',
  }[variant] as TextColor;
  const inactive = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityHint={accessibilityHint}
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        styles[size],
        styles[variant],
        fullWidth && styles.fullWidth,
        inactive && styles.disabled,
        pressed && styles.pressed,
        style,
      ]}
    >
      {loading ? (
        <ActivityIndicator color={colors[labelColor]} size="small" style={{ marginRight: 8 }} />
      ) : null}
      <Text variant="label" color={labelColor} weight="700" style={{ fontSize: 16 }}>
        {label}
      </Text>
    </Pressable>
  );
}
