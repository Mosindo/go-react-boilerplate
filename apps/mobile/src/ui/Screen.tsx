import React from "react";
import { Pressable, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = {
  title?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  scroll?: boolean;
  children: React.ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
  /** Tab roots sit above the tab bar, which already handles the bottom inset. */
  noBottomInset?: boolean;
};

export function Screen({ title, onBack, right, scroll, children, contentStyle, noBottomInset }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const body = scroll ? (
    <ScrollView contentContainerStyle={[styles.scroll, contentStyle]} keyboardShouldPersistTaps="handled">
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.flex, contentStyle]}>{children}</View>
  );
  return (
    <View
      style={[
        styles.flex,
        { backgroundColor: colors.background, paddingTop: insets.top, paddingBottom: noBottomInset ? 0 : insets.bottom }
      ]}
    >
      {title || onBack || right ? (
        <View style={styles.header}>
          {onBack ? (
            <Pressable accessibilityRole="button" accessibilityLabel="Retour" onPress={onBack} hitSlop={12} style={styles.back}>
              <Text variant="heading">‹</Text>
            </Pressable>
          ) : null}
          <Text variant="heading" style={styles.title} numberOfLines={1}>
            {title}
          </Text>
          <View>{right}</View>
        </View>
      ) : null}
      {body}
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  scroll: { padding: 20, gap: 16, paddingBottom: 40 },
  header: { flexDirection: "row", alignItems: "center", paddingHorizontal: 16, minHeight: 52, gap: 8 },
  back: { width: 32, height: 40, justifyContent: "center" },
  title: { flex: 1 }
});
