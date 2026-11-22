import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { colors, layout, spacing } from "./theme";

type ScreenProps = {
  children: ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  footer?: ReactNode;
  testID?: string;
  contentStyle?: StyleProp<ViewStyle>;
};

/** Safe-area aware page with an optional scroll container and sticky footer. */
export function Screen({ children, scroll, edges = ["top", "left", "right"], padded = true, footer, testID, contentStyle }: ScreenProps) {
  const inner = [styles.content, padded ? styles.padded : null, contentStyle];
  return (
    <SafeAreaView edges={edges} style={styles.safe} testID={testID}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {scroll ? (
          <ScrollView contentContainerStyle={[inner, styles.scrollGrow]} keyboardShouldPersistTaps="handled">
            {children}
          </ScrollView>
        ) : (
          <View style={[inner, styles.flex]}>{children}</View>
        )}
        {footer ? <View style={styles.footer}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: colors.background
  },
  flex: {
    flex: 1
  },
  content: {
    width: "100%",
    maxWidth: layout.maxContentWidth,
    alignSelf: "center"
  },
  padded: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md
  },
  scrollGrow: {
    flexGrow: 1,
    paddingBottom: spacing.xxl
  },
  footer: {
    width: "100%",
    maxWidth: layout.maxContentWidth,
    alignSelf: "center",
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: spacing.lg,
    gap: spacing.sm
  }
});
