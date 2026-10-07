import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text, spacing } from "../../shared/ui";
import { usePalette } from "./palette";

type Props = {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  right?: ReactNode;
  leading?: ReactNode;
  onTitlePress?: () => void;
  testID?: string;
};

export function ScreenHeader({ title, subtitle, onBack, right, leading, onTitlePress, testID }: Props) {
  const p = usePalette();
  return (
    <View
      style={[styles.row, { borderBottomColor: p.border, backgroundColor: p.background }]}
      testID={testID}
    >
      {onBack ? (
        <Pressable
          accessibilityLabel="Go back"
          accessibilityRole="button"
          hitSlop={10}
          onPress={onBack}
          style={styles.back}
          testID="header-back"
        >
          <Text style={styles.backGlyph} weight="bold">
            {"‹"}
          </Text>
        </Pressable>
      ) : null}
      {leading}
      <Pressable
        accessibilityRole={onTitlePress ? "button" : "header"}
        disabled={!onTitlePress}
        onPress={onTitlePress}
        style={styles.titleWrap}
      >
        <Text numberOfLines={1} variant="heading" weight="bold">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </Pressable>
      {right}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  back: { width: 36, height: 40, alignItems: "center", justifyContent: "center" },
  backGlyph: { fontSize: 32, lineHeight: 36 },
  titleWrap: { flex: 1, justifyContent: "center" }
});
