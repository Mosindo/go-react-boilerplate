import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { Animated, Modal, Pressable, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { fr } from '../i18n';
import { USE_NATIVE_DRIVER } from '../lib/motion';
import { MAX_CONTENT_WIDTH, useStyles } from '../theme';
import { Button } from './Button';
import type { ButtonVariant } from './Button';
import { Text } from './Text';

interface SheetProps {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
}

/** Bottom sheet with backdrop. Keeps itself mounted until the exit animation finishes. */
export function Sheet({ visible, onClose, title, children }: SheetProps) {
  const insets = useSafeAreaInsets();
  const progress = useRef(new Animated.Value(0)).current;
  const [mounted, setMounted] = useState(visible);
  const styles = useStyles((t) => ({
    backdrop: {
      flex: 1,
      backgroundColor: t.colors.overlay,
      justifyContent: 'flex-end',
      alignItems: 'center',
    },
    sheet: {
      width: '100%',
      maxWidth: MAX_CONTENT_WIDTH,
      backgroundColor: t.colors.surface,
      borderTopLeftRadius: t.radius.xl,
      borderTopRightRadius: t.radius.xl,
      paddingTop: t.spacing.md,
      paddingHorizontal: t.spacing.lg,
      gap: t.spacing.md,
    },
    grabber: {
      alignSelf: 'center',
      width: 40,
      height: 4,
      borderRadius: 2,
      backgroundColor: t.colors.border,
    },
  }));

  useEffect(() => {
    if (visible) {
      setMounted(true);
      Animated.timing(progress, {
        toValue: 1,
        duration: 220,
        useNativeDriver: USE_NATIVE_DRIVER,
      }).start();
    } else {
      Animated.timing(progress, {
        toValue: 0,
        duration: 160,
        useNativeDriver: USE_NATIVE_DRIVER,
      }).start(({ finished }) => finished && setMounted(false));
    }
  }, [visible, progress]);

  return (
    <Modal
      transparent
      visible={mounted}
      animationType="none"
      onRequestClose={onClose}
      statusBarTranslucent
    >
      <Animated.View style={[styles.backdrop, { opacity: progress }]}>
        <Pressable
          style={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 }}
          accessibilityRole="button"
          accessibilityLabel={fr.common.close}
          onPress={onClose}
        />
        <Animated.View
          accessibilityViewIsModal
          style={[
            styles.sheet,
            { paddingBottom: insets.bottom + 16 },
            {
              transform: [
                { translateY: progress.interpolate({ inputRange: [0, 1], outputRange: [60, 0] }) },
              ],
            },
          ]}
        >
          <View style={styles.grabber} />
          {title ? (
            <Text variant="heading" align="center">
              {title}
            </Text>
          ) : null}
          {children}
        </Animated.View>
      </Animated.View>
    </Modal>
  );
}

export interface SheetAction {
  label: string;
  onPress: () => void;
  variant?: ButtonVariant;
}

/** Sheet containing a vertical list of actions plus a cancel button. */
export function ActionSheet({
  visible,
  onClose,
  title,
  actions,
}: {
  visible: boolean;
  onClose: () => void;
  title?: string;
  actions: SheetAction[];
}) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title}>
      {actions.map((action) => (
        <Button
          key={action.label}
          label={action.label}
          variant={action.variant ?? 'ghost'}
          onPress={() => {
            onClose();
            action.onPress();
          }}
        />
      ))}
      <Button label={fr.common.cancel} variant="soft" onPress={onClose} />
    </Sheet>
  );
}
