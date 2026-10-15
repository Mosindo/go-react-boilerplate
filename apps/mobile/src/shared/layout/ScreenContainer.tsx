import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, RefreshControl, ScrollView, View, type StyleProp, type ViewStyle } from "react-native";
import { type Edge } from "react-native-safe-area-context";
import { useTheme } from "../../theme";
import { spacing } from "../ui";
import { SafeAreaLayout } from "./SafeAreaLayout";

type ScreenContainerProps = {
  children: ReactNode;
  /** Wraps content in a ScrollView. Leave off for screens that own a FlatList. */
  scroll?: boolean;
  padded?: boolean;
  /** Pads and lifts content above the keyboard (forms). */
  keyboardAware?: boolean;
  edges?: Edge[];
  refreshing?: boolean;
  onRefresh?: () => void;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function ScreenContainer({
  children,
  contentStyle,
  edges = ["top", "left", "right"],
  keyboardAware = false,
  onRefresh,
  padded = true,
  refreshing = false,
  scroll = false,
  testID
}: ScreenContainerProps) {
  const palette = useTheme();
  const padding: ViewStyle = padded ? { paddingHorizontal: spacing.lg } : {};

  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[{ flexGrow: 1, paddingBottom: spacing.xxl, gap: spacing.lg }, padding, contentStyle]}
      keyboardDismissMode="on-drag"
      keyboardShouldPersistTaps="handled"
      refreshControl={onRefresh ? <RefreshControl onRefresh={onRefresh} refreshing={refreshing} tintColor={palette.primary} /> : undefined}
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[{ flex: 1 }, padding, contentStyle]}>{children}</View>
  );

  return (
    <SafeAreaLayout edges={edges}>
      <View style={{ flex: 1 }} testID={testID}>
        {keyboardAware ? (
          <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={{ flex: 1 }}>
            {body}
          </KeyboardAvoidingView>
        ) : (
          body
        )}
      </View>
    </SafeAreaLayout>
  );
}
