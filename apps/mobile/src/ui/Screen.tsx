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
import { SafeAreaView } from "react-native-safe-area-context";
import { useTheme } from "../theme";

type Props = {
  children: ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: ("top" | "bottom")[];
  style?: StyleProp<ViewStyle>;
};

/** Themed, safe-area aware screen wrapper. Scrolls and avoids the keyboard when asked. */
export function Screen({ children, scroll, padded = true, edges = ["bottom"], style }: Props) {
  const t = useTheme();
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[padded && styles.padded, styles.grow, style]}
      keyboardShouldPersistTaps="handled"
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.grow, padded && styles.padded, style]}>{children}</View>
  );
  return (
    <SafeAreaView style={[styles.grow, { backgroundColor: t.bg }]} edges={edges}>
      <KeyboardAvoidingView
        style={styles.grow}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        {body}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  grow: { flexGrow: 1, flex: 1 },
  padded: { padding: 20, gap: 16 }
});
