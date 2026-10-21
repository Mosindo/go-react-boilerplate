import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { MIN_TOUCH, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = { label: string; value: string; onDecrement: () => void; onIncrement: () => void; testID?: string };

export function Stepper({ label, value, onDecrement, onIncrement, testID }: Props) {
  const theme = useTheme();
  const btn = (icon: "remove" | "add", action: () => void, name: string) => (
    <Pressable
      testID={testID ? `${testID}-${name}` : undefined}
      accessibilityRole="button"
      accessibilityLabel={`${name === "dec" ? "Diminuer" : "Augmenter"} ${label}`}
      onPress={action}
      style={[styles.btn, { borderColor: theme.border, backgroundColor: theme.surface }]}
    >
      <Ionicons name={icon} size={20} color={theme.text} />
    </Pressable>
  );
  return (
    <View style={styles.row}>
      <Text variant="label" style={{ flex: 1 }}>
        {label}
      </Text>
      {btn("remove", onDecrement, "dec")}
      <Text testID={testID ? `${testID}-value` : undefined} style={styles.value} center>
        {value}
      </Text>
      {btn("add", onIncrement, "inc")}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  btn: { width: MIN_TOUCH, height: MIN_TOUCH, borderRadius: radius.pill, borderWidth: 1, alignItems: "center", justifyContent: "center" },
  value: { minWidth: 76, fontWeight: "700" }
});
