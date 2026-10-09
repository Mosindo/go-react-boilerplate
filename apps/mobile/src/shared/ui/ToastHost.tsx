import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { radius, spacing, useTheme, MAX_CONTENT_WIDTH } from '../../theme';
import { toast, useToasts } from '../toastStore';
import { Text } from './Text';

export function ToastHost() {
  const items = useToasts();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  if (items.length === 0) return null;
  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.sm }]}>
      {items.map((t) => {
        const bg =
          t.kind === 'error'
            ? colors.danger
            : t.kind === 'success'
              ? colors.success
              : colors.accent;
        const fg =
          t.kind === 'error' ? colors.onDanger : t.kind === 'success' ? '#FFFFFF' : colors.onAccent;
        return (
          <Pressable
            key={t.id}
            testID={`toast-${t.kind}`}
            accessibilityRole="alert"
            accessibilityLiveRegion="polite"
            accessibilityLabel={t.message}
            onPress={() => toast.dismiss(t.id)}
            style={[styles.toast, { backgroundColor: bg }]}
          >
            <Text variant="label" style={[styles.msg, { color: fg }]}>
              {t.message}
            </Text>
            {t.action ? (
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={t.action.label}
                testID="toast-action"
                hitSlop={8}
                onPress={() => {
                  toast.dismiss(t.id);
                  t.action?.onPress();
                }}
              >
                <Text variant="label" style={[styles.action, { color: fg }]}>
                  {t.action.label}
                </Text>
              </Pressable>
            ) : null}
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  host: {
    position: 'absolute',
    left: spacing.lg,
    right: spacing.lg,
    alignItems: 'center',
    zIndex: 1000,
  },
  toast: {
    width: '100%',
    maxWidth: MAX_CONTENT_WIDTH - 2 * spacing.lg,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderRadius: radius.md,
    marginBottom: spacing.sm,
    shadowColor: '#000',
    shadowOpacity: 0.2,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 4 },
    elevation: 6,
  },
  msg: { flex: 1 },
  action: { marginLeft: spacing.md, textDecorationLine: 'underline' },
});
