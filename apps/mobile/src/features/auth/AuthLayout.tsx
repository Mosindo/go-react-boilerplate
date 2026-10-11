import type { ReactNode } from 'react';
import { View } from 'react-native';
import { fr } from '../../i18n';
import { useStyles } from '../../theme';
import { Screen, Text, icons } from '../../ui';

interface AuthLayoutProps {
  title: string;
  subtitle?: string;
  children: ReactNode;
}

export function AuthLayout({ title, subtitle, children }: AuthLayoutProps) {
  const styles = useStyles((t) => ({
    brand: { flexDirection: 'row', alignItems: 'center', gap: t.spacing.sm },
    header: { gap: t.spacing.sm, marginBottom: t.spacing.sm },
    form: { gap: t.spacing.lg },
  }));
  return (
    <Screen scroll edges={['bottom']}>
      <View style={styles.header}>
        <View style={styles.brand} accessible accessibilityLabel={fr.brand.name}>
          <Text color="primary" style={{ fontSize: 22 }} accessible={false}>
            {icons.spark}
          </Text>
          <Text variant="label" color="primary" weight="800" accessible={false}>
            {fr.brand.name}
          </Text>
        </View>
        <Text variant="title">{title}</Text>
        {subtitle ? <Text color="textMuted">{subtitle}</Text> : null}
      </View>
      <View style={styles.form}>{children}</View>
    </Screen>
  );
}
