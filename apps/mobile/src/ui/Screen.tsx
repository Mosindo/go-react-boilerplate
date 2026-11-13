import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import React from "react";
import {
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: Edge[];
  footer?: React.ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Safe-area aware page with optional scrolling and a sticky footer. */
export function Screen({
  children,
  scroll,
  padded = true,
  edges = ["top", "bottom"],
  footer,
  contentStyle,
  testID
}: ScreenProps) {
  const { colors, spacing } = useTheme();
  const padding = padded ? { paddingHorizontal: spacing.lg } : null;
  return (
    <SafeAreaView edges={edges} style={[styles.flex, { backgroundColor: colors.background }]} testID={testID}>
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {scroll ? (
          <ScrollView
            style={styles.flex}
            contentContainerStyle={[padding, { paddingBottom: spacing.xxl }, contentStyle]}
            keyboardShouldPersistTaps="handled"
          >
            {children}
          </ScrollView>
        ) : (
          <View style={[styles.flex, padding, contentStyle]}>{children}</View>
        )}
        {footer ? <View style={[padding, { paddingVertical: spacing.sm }]}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

type HeaderProps = {
  title?: string;
  back?: boolean;
  right?: React.ReactNode;
  children?: React.ReactNode;
};

export function Header({ title, back, right, children }: HeaderProps) {
  const navigation = useNavigation();
  const { colors, spacing } = useTheme();
  return (
    <View style={[styles.header, { paddingVertical: spacing.sm }]}>
      <View style={styles.headerSide}>
        {back ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Retour"
            hitSlop={12}
            onPress={() => navigation.goBack()}
            testID="header-back"
          >
            <Ionicons name="chevron-back" size={26} color={colors.text} />
          </Pressable>
        ) : null}
      </View>
      <View style={styles.headerCenter}>
        {children ?? (
          <Text variant="heading" numberOfLines={1} accessibilityRole="header">
            {title}
          </Text>
        )}
      </View>
      <View style={[styles.headerSide, styles.headerRight]}>{right}</View>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center", minHeight: 52 },
  headerSide: { width: 44, justifyContent: "center" },
  headerRight: { alignItems: "flex-end" },
  headerCenter: { flex: 1, alignItems: "center" }
});
