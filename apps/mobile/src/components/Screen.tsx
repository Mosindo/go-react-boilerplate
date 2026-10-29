import React from "react";
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  /** Edges padded by the safe area. Screens under a native header pass ["bottom"]. */
  edges?: Edge[];
  padded?: boolean;
  style?: StyleProp<ViewStyle>;
  contentStyle?: StyleProp<ViewStyle>;
};

export function Screen({
  children,
  scroll = false,
  edges = ["top", "bottom"],
  padded = true,
  style,
  contentStyle
}: Props) {
  const theme = useTheme();
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[padded && styles.padded, styles.grow, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.grow, padded && styles.padded, contentStyle]}>{children}</View>
  );

  return (
    <SafeAreaView edges={edges} style={[styles.grow, { backgroundColor: theme.background }, style]}>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        style={styles.grow}
      >
        {body}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  grow: { flexGrow: 1, flex: 1 },
  padded: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md }
});
