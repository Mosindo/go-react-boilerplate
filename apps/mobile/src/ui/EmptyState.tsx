import { View } from 'react-native';
import { useStyles } from '../theme';
import { Button } from './Button';
import { Text } from './Text';

interface EmptyStateProps {
  glyph: string;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({ glyph, title, message, actionLabel, onAction }: EmptyStateProps) {
  const styles = useStyles((t) => ({
    wrap: {
      alignItems: 'center',
      justifyContent: 'center',
      padding: t.spacing.xxl,
      gap: t.spacing.md,
      flexGrow: 1,
    },
    badge: {
      width: 84,
      height: 84,
      borderRadius: 42,
      backgroundColor: t.colors.primarySoft,
      alignItems: 'center',
      justifyContent: 'center',
      marginBottom: t.spacing.sm,
    },
    action: { marginTop: t.spacing.md, alignSelf: 'stretch' },
  }));
  return (
    <View style={styles.wrap} accessibilityRole="summary">
      <View style={styles.badge} accessible={false}>
        <Text color="primary" style={{ fontSize: 36, lineHeight: 44 }}>
          {glyph}
        </Text>
      </View>
      <Text variant="heading" align="center">
        {title}
      </Text>
      {message ? (
        <Text color="textMuted" align="center">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? (
        <View style={styles.action}>
          <Button label={actionLabel} onPress={onAction} variant="soft" />
        </View>
      ) : null}
    </View>
  );
}
