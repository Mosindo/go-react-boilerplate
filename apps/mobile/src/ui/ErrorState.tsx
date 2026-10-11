import { View } from 'react-native';
import { fr } from '../i18n';
import { useStyles } from '../theme';
import { Button } from './Button';
import { icons } from './icons';
import { Text } from './Text';

interface ErrorStateProps {
  message: string;
  onRetry?: () => void;
  title?: string;
}

export function ErrorState({ message, onRetry, title = fr.common.errorTitle }: ErrorStateProps) {
  const styles = useStyles((t) => ({
    wrap: {
      alignItems: 'center',
      justifyContent: 'center',
      padding: t.spacing.xxl,
      gap: t.spacing.md,
      flexGrow: 1,
    },
    badge: {
      width: 72,
      height: 72,
      borderRadius: 36,
      backgroundColor: t.colors.dangerSoft,
      alignItems: 'center',
      justifyContent: 'center',
    },
    action: { marginTop: t.spacing.sm, alignSelf: 'stretch' },
  }));
  return (
    <View style={styles.wrap} accessibilityRole="alert">
      <View style={styles.badge} accessible={false}>
        <Text color="danger" style={{ fontSize: 30, lineHeight: 38 }}>
          {icons.info}
        </Text>
      </View>
      <Text variant="heading" align="center">
        {title}
      </Text>
      <Text color="textMuted" align="center">
        {message}
      </Text>
      {onRetry ? (
        <View style={styles.action}>
          <Button label={fr.common.retry} onPress={onRetry} variant="ghost" />
        </View>
      ) : null}
    </View>
  );
}
