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
import { SafeAreaView } from "react-native-safe-area-context";
import { spacing, useTheme } from "../theme";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: ("top" | "bottom" | "left" | "right")[];
  contentStyle?: StyleProp<ViewStyle>;
};

/** Page container: themed background, safe areas, optional scrolling and keyboard handling. */
export function Screen({
  children,
  scroll,
  padded = true,
  edges = ["top", "left", "right"],
  contentStyle
}: Props) {
  const theme = useTheme();
  const inner = scroll ? (
    <ScrollView
      keyboardShouldPersistTaps="handled"
      contentContainerStyle={[padded ? styles.padded : null, styles.grow, contentStyle]}
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.grow, padded ? styles.padded : null, contentStyle]}>{children}</View>
  );

  return (
    <SafeAreaView edges={edges} style={[styles.grow, { backgroundColor: theme.bg }]}>
      <KeyboardAvoidingView style={styles.grow} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {inner}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  grow: { flexGrow: 1, flex: 1 },
  padded: { padding: spacing.lg, gap: spacing.lg }
});
