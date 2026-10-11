import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { ReactNode } from 'react';
import { Animated, Pressable, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { USE_NATIVE_DRIVER } from '../lib/motion';
import { tempId } from '../lib/uniqueId';
import { useStyles, useTheme } from '../theme';
import { Text } from './Text';

export type ToastKind = 'success' | 'error' | 'info';

interface ToastItem {
  id: string;
  message: string;
  kind: ToastKind;
  actionLabel?: string;
  onAction?: () => void;
}

interface ToastApi {
  show(message: string, kind?: ToastKind, action?: { label: string; onPress: () => void }): void;
  success(message: string): void;
  error(message: string): void;
}

const ToastContext = createContext<ToastApi | null>(null);

const DURATION_MS = 3500;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<ToastItem | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const dismiss = useCallback(() => {
    if (timer.current) clearTimeout(timer.current);
    setToast(null);
  }, []);

  const show = useCallback<ToastApi['show']>((message, kind = 'info', action) => {
    if (timer.current) clearTimeout(timer.current);
    setToast({
      id: tempId('toast'),
      message,
      kind,
      actionLabel: action?.label,
      onAction: action?.onPress,
    });
    timer.current = setTimeout(() => setToast(null), DURATION_MS);
  }, []);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  const api = useMemo<ToastApi>(
    () => ({
      show,
      success: (message) => show(message, 'success'),
      error: (message) => show(message, 'error'),
    }),
    [show],
  );

  return (
    <ToastContext.Provider value={api}>
      {children}
      {toast ? <ToastView key={toast.id} toast={toast} onDismiss={dismiss} /> : null}
    </ToastContext.Provider>
  );
}

function ToastView({ toast, onDismiss }: { toast: ToastItem; onDismiss: () => void }) {
  const insets = useSafeAreaInsets();
  const { colors } = useTheme();
  const progress = useRef(new Animated.Value(0)).current;
  const styles = useStyles((t) => ({
    wrap: { position: 'absolute', left: t.spacing.lg, right: t.spacing.lg, alignItems: 'center' },
    toast: {
      flexDirection: 'row',
      alignItems: 'center',
      gap: t.spacing.md,
      maxWidth: 440,
      paddingVertical: t.spacing.md,
      paddingHorizontal: t.spacing.lg,
      borderRadius: t.radius.md,
      backgroundColor: t.colors.text,
      shadowColor: '#000',
      shadowOpacity: 0.2,
      shadowRadius: 12,
      shadowOffset: { width: 0, height: 4 },
      elevation: 6,
    },
    dot: { width: 8, height: 8, borderRadius: 4 },
    action: { minHeight: 44, justifyContent: 'center', paddingLeft: t.spacing.sm },
  }));
  useEffect(() => {
    Animated.timing(progress, {
      toValue: 1,
      duration: 220,
      useNativeDriver: USE_NATIVE_DRIVER,
    }).start();
  }, [progress]);

  const dotColor =
    toast.kind === 'success'
      ? colors.success
      : toast.kind === 'error'
        ? colors.danger
        : colors.primary;
  return (
    <Animated.View
      pointerEvents="box-none"
      style={[
        styles.wrap,
        { bottom: insets.bottom + 84 },
        {
          opacity: progress,
          transform: [
            { translateY: progress.interpolate({ inputRange: [0, 1], outputRange: [16, 0] }) },
          ],
        },
      ]}
    >
      <Pressable
        onPress={onDismiss}
        accessibilityRole="alert"
        accessibilityLiveRegion="polite"
        accessibilityLabel={toast.message}
        style={styles.toast}
      >
        <View style={[styles.dot, { backgroundColor: dotColor }]} />
        <Text variant="small" style={{ flexShrink: 1, color: colors.bg }}>
          {toast.message}
        </Text>
        {toast.actionLabel ? (
          <Pressable
            accessibilityRole="button"
            onPress={() => {
              toast.onAction?.();
              onDismiss();
            }}
            style={styles.action}
          >
            <Text variant="label" style={{ color: colors.primarySoft }}>
              {toast.actionLabel}
            </Text>
          </Pressable>
        ) : null}
      </Pressable>
    </Animated.View>
  );
}

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used inside <ToastProvider>');
  return ctx;
}
