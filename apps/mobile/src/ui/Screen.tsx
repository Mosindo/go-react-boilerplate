import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { spacing, useTheme } from "../theme";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  /** Apply the top safe-area inset (screens without a native header). */
  top?: boolean;
  padded?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Screen({ children, scroll = false, top = true, padded = true, style }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const container: ViewStyle = { flex: 1, backgroundColor: colors.background, paddingTop: top ? insets.top : 0 };
  const content: ViewStyle = { padding: padded ? spacing.lg : 0, gap: spacing.lg, paddingBottom: spacing.xxl };

  return (
    <KeyboardAvoidingView style={container} behavior={Platform.OS === "ios" ? "padding" : undefined}>
      {scroll ? (
        <ScrollView contentContainerStyle={[content, style]} keyboardShouldPersistTaps="handled" showsVerticalScrollIndicator={false}>
          {children}
        </ScrollView>
      ) : (
        <View style={[styles.flex, padded ? { padding: spacing.lg } : null, style]}>{children}</View>
      )}
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({ flex: { flex: 1 } });
