import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { makeStyles } from "../theme";

type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  keyboard?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  footer?: React.ReactNode;
};

export function Screen({ children, scroll, edges = ["top", "left", "right"], padded = true, keyboard, contentStyle, footer }: ScreenProps) {
  const styles = useStyles();
  const body = scroll ? (
    <ScrollView
      keyboardShouldPersistTaps="handled"
      contentContainerStyle={[padded ? styles.padded : null, styles.scrollContent, contentStyle]}
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.flex, padded ? styles.padded : null, contentStyle]}>{children}</View>
  );

  return (
    <SafeAreaView edges={edges} style={styles.root}>
      {keyboard ? (
        <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
          <View style={styles.inner}>
            {body}
            {footer ? <View style={styles.footer}>{footer}</View> : null}
          </View>
        </KeyboardAvoidingView>
      ) : (
        <View style={styles.inner}>
          {body}
          {footer ? <View style={styles.footer}>{footer}</View> : null}
        </View>
      )}
    </SafeAreaView>
  );
}

const useStyles = makeStyles((t) => ({
  root: { flex: 1, backgroundColor: t.colors.background },
  flex: { flex: 1 },
  inner: { flex: 1, width: "100%", maxWidth: 640, alignSelf: "center" },
  padded: { paddingHorizontal: t.spacing.xl, paddingTop: t.spacing.lg },
  scrollContent: { paddingBottom: t.spacing.xxxl, gap: t.spacing.lg },
  footer: { paddingHorizontal: t.spacing.xl, paddingTop: t.spacing.md, paddingBottom: t.spacing.lg, gap: t.spacing.sm }
}));
