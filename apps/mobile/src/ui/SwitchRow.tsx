import { Switch, View } from 'react-native';
import { useStyles, useTheme } from '../theme';
import { Text } from './Text';

interface SwitchRowProps {
  label: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
}

export function SwitchRow({
  label,
  description,
  value,
  onValueChange,
  disabled,
  testID,
}: SwitchRowProps) {
  const { colors } = useTheme();
  const styles = useStyles((t) => ({
    row: {
      minHeight: 56,
      flexDirection: 'row',
      alignItems: 'center',
      gap: t.spacing.md,
      padding: t.spacing.lg,
    },
    text: { flex: 1, gap: 2 },
  }));
  return (
    <View style={styles.row}>
      <View style={styles.text}>
        <Text weight="600">{label}</Text>
        {description ? (
          <Text variant="small" color="textMuted">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        testID={testID}
        accessibilityRole="switch"
        value={value}
        disabled={disabled}
        onValueChange={onValueChange}
        trackColor={{ true: colors.primary, false: colors.border }}
        thumbColor="#FFFFFF"
      />
    </View>
  );
}
