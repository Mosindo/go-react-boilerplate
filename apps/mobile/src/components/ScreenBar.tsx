import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { Text, spacing } from "../shared/ui";

type ScreenBarProps = { title: string; right?: ReactNode };

/** Top bar for pushed screens: back button, title, optional action. */
export function ScreenBar({ title, right }: ScreenBarProps) {
  const navigation = useNavigation();
  return (
    <View style={styles.bar}>
      <Pressable
        accessibilityLabel="Retour"
        accessibilityRole="button"
        onPress={() => navigation.goBack()}
        style={styles.back}
        testID="screen-back"
      >
        <Text variant="heading">←</Text>
      </Pressable>
      <Text accessibilityRole="header" style={styles.title} variant="heading" weight="bold">
        {title}
      </Text>
      <View style={styles.right}>{right}</View>
    </View>
  );
}

const styles = StyleSheet.create({
  bar: { flexDirection: "row", alignItems: "center", gap: spacing.sm, marginBottom: spacing.lg },
  back: { width: 44, height: 44, alignItems: "center", justifyContent: "center", marginLeft: -spacing.sm },
  title: { flex: 1 },
  right: { minWidth: 44, alignItems: "flex-end" }
});
