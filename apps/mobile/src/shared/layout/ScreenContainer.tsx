import type { ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import type { Edge } from "react-native-safe-area-context";
import { layout, spacing } from "../../theme";
import { SafeAreaLayout } from "./SafeAreaLayout";

/**
 * Standard screen shell: safe area, themed background, content capped to a readable width on tablets and
 * the web, optional scrolling and keyboard avoidance for forms.
 */
export function ScreenContainer({
  children,
  scroll,
  padded = true,
  edges = ["top", "bottom"],
  contentStyle,
  testID
}: {
  children: ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: Edge[];
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
}) {
  const inner = [styles.content, padded && styles.padded, contentStyle];
  return (
    <SafeAreaLayout edges={edges}>
      <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined} testID={testID}>
        {scroll ? (
          <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled" showsVerticalScrollIndicator={false}>
            <View style={inner}>{children}</View>
          </ScrollView>
        ) : (
          <View style={[styles.fill, styles.center]}>
            <View style={[styles.fill, inner]}>{children}</View>
          </View>
        )}
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  center: { alignItems: "center" },
  scroll: { flexGrow: 1, alignItems: "center" },
  content: { width: "100%", maxWidth: layout.maxContentWidth },
  padded: { paddingHorizontal: spacing.lg, paddingVertical: spacing.lg, gap: spacing.lg }
});
