import React from "react";
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { spacing, useTheme } from "../theme/theme";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: ("top" | "bottom" | "left" | "right")[];
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({
  children,
  scroll,
  padded = true,
  edges = ["top", "bottom"],
  contentStyle,
  testID,
}: Props) {
  const { colors } = useTheme();
  const inner = scroll ? (
    <ScrollView
      keyboardShouldPersistTaps="handled"
      contentContainerStyle={[padded && styles.padded, styles.grow, contentStyle]}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.grow, padded && styles.padded, contentStyle]}>{children}</View>
  );
  return (
    <SafeAreaView
      testID={testID}
      edges={edges}
      style={[styles.grow, { backgroundColor: colors.bg }]}
    >
      <KeyboardAvoidingView
        style={styles.grow}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={Platform.OS === "ios" ? 8 : 0}
      >
        {inner}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  grow: { flexGrow: 1, flex: 1 },
  padded: { padding: spacing.lg, gap: spacing.lg },
});
