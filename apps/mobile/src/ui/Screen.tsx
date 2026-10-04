import React, { type ReactNode } from "react";
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
import { spacing, useColors } from "../theme";

type ScreenProps = {
  children: ReactNode;
  scroll?: boolean;
  /** Safe-area edges to pad. Screens under a navigation header should omit "top". */
  edges?: Edge[];
  padded?: boolean;
  keyboardAvoiding?: boolean;
  style?: StyleProp<ViewStyle>;
  contentStyle?: StyleProp<ViewStyle>;
};

export function Screen({
  children,
  scroll = false,
  edges = ["left", "right", "bottom"],
  padded = true,
  keyboardAvoiding = false,
  style,
  contentStyle
}: ScreenProps) {
  const colors = useColors();
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[padded && styles.padded, styles.scrollContent, contentStyle]}
      keyboardShouldPersistTaps="handled"
    >
      <View style={styles.maxWidth}>{children}</View>
    </ScrollView>
  ) : (
    <View style={[styles.flex, padded && styles.padded, contentStyle]}>{children}</View>
  );

  return (
    <SafeAreaView edges={edges} style={[styles.flex, { backgroundColor: colors.background }, style]}>
      {keyboardAvoiding ? (
        <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
          {body}
        </KeyboardAvoidingView>
      ) : (
        body
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  padded: { paddingHorizontal: spacing.md, paddingVertical: spacing.md },
  scrollContent: { flexGrow: 1 },
  maxWidth: { width: "100%", maxWidth: 560, alignSelf: "center", gap: spacing.md }
});
