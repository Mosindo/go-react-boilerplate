import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { colors, layout, spacing } from "../theme";

type ScreenProps = {
  children: ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  footer?: ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({ children, scroll = false, edges = ["top", "left", "right"], footer, contentStyle, testID }: ScreenProps) {
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[styles.scrollContent, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      <View style={styles.constrained}>{children}</View>
    </ScrollView>
  ) : (
    <View style={[styles.fill, styles.padded, contentStyle]}>
      <View style={[styles.constrained, styles.fill]}>{children}</View>
    </View>
  );

  return (
    <SafeAreaView edges={edges} style={styles.safe} testID={testID}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.fill}>
        {body}
        {footer ? (
          <View style={styles.footer}>
            <View style={styles.constrained}>{footer}</View>
          </View>
        ) : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: colors.bg
  },
  fill: {
    flex: 1
  },
  padded: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md
  },
  scrollContent: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: spacing.xxxl
  },
  constrained: {
    width: "100%",
    maxWidth: layout.maxContentWidth,
    alignSelf: "center"
  },
  footer: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: spacing.lg,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border,
    backgroundColor: colors.bg
  }
});
