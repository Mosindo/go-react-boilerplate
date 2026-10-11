import { useEffect, useRef } from 'react';
import { Animated, View } from 'react-native';
import { fr } from '../../i18n';
import { USE_NATIVE_DRIVER } from '../../lib/motion';
import type { AuthScreenProps } from '../../navigation/types';
import { useStyles } from '../../theme';
import { Button, Screen, Text, icons } from '../../ui';

export function WelcomeScreen({ navigation }: AuthScreenProps<'Welcome'>) {
  const glow = useRef(new Animated.Value(0)).current;
  const enter = useRef(new Animated.Value(0)).current;
  const styles = useStyles((t) => ({
    hero: {
      flex: 1,
      alignItems: 'center',
      justifyContent: 'center',
      gap: t.spacing.lg,
      paddingVertical: t.spacing.xxl,
    },
    flame: {
      width: 120,
      height: 120,
      borderRadius: 60,
      backgroundColor: t.colors.primarySoft,
      alignItems: 'center',
      justifyContent: 'center',
    },
    actions: { gap: t.spacing.md, paddingBottom: t.spacing.lg },
  }));

  useEffect(() => {
    Animated.timing(enter, {
      toValue: 1,
      duration: 600,
      useNativeDriver: USE_NATIVE_DRIVER,
    }).start();
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(glow, { toValue: 1, duration: 1800, useNativeDriver: USE_NATIVE_DRIVER }),
        Animated.timing(glow, { toValue: 0, duration: 1800, useNativeDriver: USE_NATIVE_DRIVER }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [enter, glow]);

  return (
    <Screen edges={['top', 'bottom']}>
      <Animated.View
        style={[
          styles.hero,
          {
            opacity: enter,
            transform: [
              { translateY: enter.interpolate({ inputRange: [0, 1], outputRange: [16, 0] }) },
            ],
          },
        ]}
      >
        <Animated.View
          style={[
            styles.flame,
            {
              transform: [
                { scale: glow.interpolate({ inputRange: [0, 1], outputRange: [1, 1.08] }) },
              ],
            },
          ]}
          accessible={false}
        >
          <Text color="primary" style={{ fontSize: 56, lineHeight: 68 }}>
            {icons.spark}
          </Text>
        </Animated.View>
        <Text variant="display" color="primary" accessibilityRole="header">
          {fr.brand.name}
        </Text>
        <Text variant="lead" color="textMuted" align="center">
          {fr.welcome.tagline}
        </Text>
        <Text variant="small" color="secondary" align="center" weight="600">
          {fr.welcome.freeBadge}
        </Text>
      </Animated.View>
      <View style={styles.actions}>
        <Button
          label={fr.welcome.register}
          onPress={() => navigation.navigate('Register')}
          testID="welcome-register"
        />
        <Button
          label={fr.welcome.login}
          variant="ghost"
          onPress={() => navigation.navigate('Login')}
          testID="welcome-login"
        />
      </View>
    </Screen>
  );
}
