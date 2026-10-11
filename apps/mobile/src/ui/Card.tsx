import { View } from 'react-native';
import type { StyleProp, ViewProps, ViewStyle } from 'react-native';
import { useStyles } from '../theme';

interface CardProps extends ViewProps {
  padded?: boolean;
  style?: StyleProp<ViewStyle>;
}

export function Card({ padded = true, style, ...rest }: CardProps) {
  const styles = useStyles((t) => ({
    card: {
      backgroundColor: t.colors.surface,
      borderRadius: t.radius.lg,
      borderWidth: 1,
      borderColor: t.colors.border,
      shadowColor: '#3A1D10',
      shadowOpacity: t.scheme === 'light' ? 0.07 : 0,
      shadowRadius: 12,
      shadowOffset: { width: 0, height: 4 },
      elevation: t.scheme === 'light' ? 2 : 0,
    },
    padded: { padding: t.spacing.lg },
  }));
  return <View style={[styles.card, padded && styles.padded, style]} {...rest} />;
}
