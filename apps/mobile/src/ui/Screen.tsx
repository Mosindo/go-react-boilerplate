import React from "react";
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  View,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { createStyles } from "../theme/ThemeProvider";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  /** Safe-area edges to pad; navigation headers already handle the top. */
  edges?: ("top" | "bottom")[];
  padded?: boolean;
  /** Disable when the screen manages its own keyboard avoidance (chat). */
  avoidKeyboard?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

const useStyles = createStyles(({ colors, spacing, contentMaxWidth }) => ({
  root: { flex: 1, backgroundColor: colors.background },
  center: { flex: 1, width: "100%", maxWidth: contentMaxWidth, alignSelf: "center" },
  padded: { paddingHorizontal: spacing.lg },
  scrollContent: { flexGrow: 1, paddingVertical: spacing.lg, gap: spacing.lg }
}));

/** Page container: themed background, centered max-width column (tablet/web), optional scrolling. */
export function Screen({
  children,
  scroll,
  edges = ["bottom"],
  padded = true,
  avoidKeyboard = true,
  contentStyle,
  testID
}: Props) {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  const safe = {
    paddingTop: edges.includes("top") ? insets.top : 0,
    paddingBottom: edges.includes("bottom") ? insets.bottom : 0
  };
  return (
    <KeyboardAvoidingView
      behavior={avoidKeyboard && Platform.OS === "ios" ? "padding" : undefined}
      style={[styles.root, safe]}
      testID={testID}
    >
      {scroll ? (
        <ScrollView
          contentContainerStyle={[
            styles.center,
            padded ? styles.padded : null,
            styles.scrollContent,
            contentStyle
          ]}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {children}
        </ScrollView>
      ) : (
        <View style={[styles.center, padded ? styles.padded : null, contentStyle]}>{children}</View>
      )}
    </KeyboardAvoidingView>
  );
}
