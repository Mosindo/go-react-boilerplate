import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text, colors, radii, spacing } from "../shared/ui";

type RowProps = { title: string; subtitle?: string; onPress: () => void; trailing?: string; testID?: string };

export function Row({ title, subtitle, onPress, trailing = "›", testID }: RowProps) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={styles.row} testID={testID}>
      <View style={styles.body}>
        <Text weight="semibold">{title}</Text>
        {subtitle ? (
          <Text tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      <Text tone="muted" variant="heading">
        {trailing}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    padding: spacing.lg,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  body: { flex: 1, gap: 2 }
});
