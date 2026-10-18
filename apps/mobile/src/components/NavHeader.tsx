import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text, colors, spacing } from "../shared/ui";

type NavHeaderProps = {
  title: string;
  onBack?: () => void;
  right?: React.ReactNode;
};

export function NavHeader({ title, onBack, right }: NavHeaderProps) {
  return (
    <View style={styles.row}>
      <View style={styles.side}>
        {onBack ? (
          <Pressable
            accessibilityLabel="Retour"
            accessibilityRole="button"
            hitSlop={10}
            onPress={onBack}
            style={styles.back}
            testID="nav-back"
          >
            <Text style={styles.arrow} weight="bold">
              ‹
            </Text>
          </Pressable>
        ) : null}
      </View>
      <Text accessibilityRole="header" numberOfLines={1} style={styles.title} variant="heading" weight="bold">
        {title}
      </Text>
      <View style={[styles.side, styles.right]}>{right}</View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    minHeight: 56,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    backgroundColor: colors.background
  },
  side: { width: 64 },
  right: { alignItems: "flex-end" },
  back: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  arrow: { fontSize: 32, lineHeight: 36 },
  title: { flex: 1, textAlign: "center" }
});
