import type { ReactNode } from 'react';
import { Pressable, View } from 'react-native';
import { useStyles } from '../theme';
import { icons } from './icons';
import { Text } from './Text';

interface ListRowProps {
  title: string;
  subtitle?: string;
  onPress: () => void;
  right?: ReactNode;
  danger?: boolean;
  hideChevron?: boolean;
  testID?: string;
}

export function ListRow({
  title,
  subtitle,
  onPress,
  right,
  danger,
  hideChevron,
  testID,
}: ListRowProps) {
  const styles = useStyles((t) => ({
    row: {
      minHeight: 56,
      flexDirection: 'row',
      alignItems: 'center',
      gap: t.spacing.md,
      paddingVertical: t.spacing.md,
      paddingHorizontal: t.spacing.lg,
    },
    pressed: { backgroundColor: t.colors.surfaceAlt },
    text: { flex: 1, gap: 2 },
  }));
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={subtitle ? `${title}, ${subtitle}` : title}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [styles.row, pressed && styles.pressed]}
    >
      <View style={styles.text}>
        <Text weight="600" color={danger ? 'danger' : 'text'}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="small" color="textMuted">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
      {hideChevron ? null : (
        <Text color="textMuted" accessible={false} style={{ fontSize: 22 }}>
          {icons.chevron}
        </Text>
      )}
    </Pressable>
  );
}
