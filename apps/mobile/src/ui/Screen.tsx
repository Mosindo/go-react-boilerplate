import type { ReactNode } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, View } from 'react-native';
import type { RefreshControlProps, StyleProp, ViewStyle } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useStyles } from '../theme';

interface ScreenProps {
  children: ReactNode;
  /** Wrap content in a ScrollView (forms, settings). Lists should manage their own scrolling. */
  scroll?: boolean;
  /** Safe-area edges to pad. Stack screens with a header use ['bottom']; tab roots use ['top']. */
  edges?: ('top' | 'bottom')[];
  padded?: boolean;
  refreshControl?: React.ReactElement<RefreshControlProps>;
  contentStyle?: StyleProp<ViewStyle>;
}

/** Themed page container with safe areas, keyboard handling and consistent gutters. */
export function Screen({
  children,
  scroll = false,
  edges = [],
  padded = true,
  refreshControl,
  contentStyle,
}: ScreenProps) {
  const insets = useSafeAreaInsets();
  const styles = useStyles((t) => ({
    root: { flex: 1, backgroundColor: t.colors.bg },
    padded: { paddingHorizontal: t.spacing.lg },
    scrollContent: { flexGrow: 1, paddingVertical: t.spacing.lg, gap: t.spacing.lg },
  }));
  const safe = {
    paddingTop: edges.includes('top') ? insets.top : 0,
    paddingBottom: edges.includes('bottom') ? insets.bottom : 0,
  };
  return (
    <KeyboardAvoidingView
      style={[styles.root, safe]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      {scroll ? (
        <ScrollView
          keyboardShouldPersistTaps="handled"
          keyboardDismissMode="on-drag"
          refreshControl={refreshControl}
          contentContainerStyle={[styles.scrollContent, padded && styles.padded, contentStyle]}
        >
          {children}
        </ScrollView>
      ) : (
        <View style={[{ flex: 1 }, padded && styles.padded, contentStyle]}>{children}</View>
      )}
    </KeyboardAvoidingView>
  );
}
