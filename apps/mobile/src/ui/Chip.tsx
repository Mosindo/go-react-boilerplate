import { Pressable } from 'react-native';
import { TOUCH_TARGET, useStyles } from '../theme';
import { Text } from './Text';

interface ChipProps {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  /** 'radio' for single choice groups, 'checkbox' for multi. */
  role?: 'radio' | 'checkbox';
  testID?: string;
}

export function Chip({
  label,
  selected = false,
  onPress,
  disabled,
  role = 'checkbox',
  testID,
}: ChipProps) {
  const styles = useStyles((t) => ({
    chip: {
      minHeight: TOUCH_TARGET,
      paddingHorizontal: t.spacing.lg,
      borderRadius: t.radius.pill,
      borderWidth: 1.5,
      borderColor: t.colors.border,
      backgroundColor: t.colors.surface,
      alignItems: 'center',
      justifyContent: 'center',
    },
    selected: { backgroundColor: t.colors.primarySoft, borderColor: t.colors.primary },
    static: {
      minHeight: 32,
      paddingHorizontal: t.spacing.md,
      backgroundColor: t.colors.surfaceAlt,
      borderColor: 'transparent',
    },
    disabled: { opacity: 0.45 },
    pressed: { opacity: 0.8 },
  }));
  const interactive = !!onPress;
  return (
    <Pressable
      accessibilityRole={interactive ? role : 'text'}
      accessibilityLabel={label}
      accessibilityState={
        interactive ? { selected, checked: selected, disabled: !!disabled } : undefined
      }
      disabled={disabled || !interactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.chip,
        !interactive && styles.static,
        selected && styles.selected,
        disabled && styles.disabled,
        pressed && styles.pressed,
      ]}
    >
      <Text variant="small" color={selected ? 'primary' : 'text'} weight={selected ? '700' : '500'}>
        {label}
      </Text>
    </Pressable>
  );
}
