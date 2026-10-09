import React from 'react';
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  type ScrollViewProps,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { spacing, useTheme } from '../../theme';

export type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  /** Which safe-area edges to pad (navigation headers/tab bars already handle the others). */
  edges?: ('top' | 'bottom')[];
  padded?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  footer?: React.ReactNode;
  testID?: string;
  refreshControl?: ScrollViewProps['refreshControl'];
};

/** Themed screen container: safe area, keyboard avoidance, optional scrolling. */
export function Screen({
  children,
  scroll = false,
  edges = [],
  padded = true,
  contentStyle,
  footer,
  testID,
  refreshControl,
}: ScreenProps) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const paddingTop = edges.includes('top') ? insets.top : 0;
  const paddingBottom = edges.includes('bottom') ? insets.bottom : 0;
  const inner: StyleProp<ViewStyle> = [padded ? styles.padded : null, contentStyle];
  return (
    <KeyboardAvoidingView
      style={[styles.flex, { backgroundColor: colors.background, paddingTop, paddingBottom }]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      testID={testID}
    >
      {scroll ? (
        <ScrollView
          style={styles.flex}
          contentContainerStyle={[styles.scrollContent, inner]}
          keyboardShouldPersistTaps="handled"
          refreshControl={refreshControl}
        >
          {children}
        </ScrollView>
      ) : (
        <View style={[styles.flex, inner]}>{children}</View>
      )}
      {footer ? <View style={styles.footer}>{footer}</View> : null}
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  padded: { paddingHorizontal: spacing.lg, paddingTop: spacing.lg },
  scrollContent: { flexGrow: 1, paddingBottom: spacing.xxl },
  footer: { paddingHorizontal: spacing.lg, paddingBottom: spacing.lg, paddingTop: spacing.sm },
});
